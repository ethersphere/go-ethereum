// Copyright 2017 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// +build linux darwin freebsd

package fuse

import (
	"bytes"
	"crypto/rand"
	"flag"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"

	"github.com/ethereum/go-ethereum/log"

	colorable "github.com/mattn/go-colorable"
)

var (
	loglevel = flag.Int("loglevel", 4, "verbosity of logs")
	rawlog   = flag.Bool("rawlog", false, "turn off terminal formatting in logs")
)

func init() {
	flag.Parse()
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(!*rawlog))))
}

type fileInfo struct {
	perm     uint64
	uid      int
	gid      int
	contents []byte
}

func createTestFilesAndUploadToSwarm(t *testing.T, api *api.Api, files map[string]fileInfo, uploadDir string, toEncrypt bool) string {
	os.RemoveAll(uploadDir)

	for fname, finfo := range files {
		actualPath := filepath.Join(uploadDir, fname)
		filePath := filepath.Dir(actualPath)

		err := os.MkdirAll(filePath, 0777)
		if err != nil {
			t.Fatalf("Error creating directory '%v' : %v", filePath, err)
		}

		fd, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(finfo.perm))
		if err1 != nil {
			t.Fatalf("Error creating file %v: %v", actualPath, err1)
		}

		fd.Write(finfo.contents)
		fd.Chown(finfo.uid, finfo.gid)
		fd.Chmod(os.FileMode(finfo.perm))
		fd.Sync()
		fd.Close()
	}

	bzzhash, err := api.Upload(uploadDir, "", toEncrypt)
	if err != nil {
		t.Fatalf("Error uploading directory %v: %vm encryption: %v", uploadDir, err, toEncrypt)
	}

	return bzzhash
}

func mountDir(t *testing.T, api *api.Api, files map[string]fileInfo, bzzHash string, mountDir string) *SwarmFS {
	os.RemoveAll(mountDir)
	os.MkdirAll(mountDir, 0777)
	swarmfs := NewSwarmFS(api)
	_, err := swarmfs.Mount(bzzHash, mountDir)
	if isFUSEUnsupportedError(err) {
		t.Skip("FUSE not supported:", err)
	} else if err != nil {
		t.Fatalf("Error mounting hash %v: %v", bzzHash, err)
	}

	found := false
	mi := swarmfs.Listmounts()
	for _, minfo := range mi {
		minfo.lock.RLock()
		if minfo.MountPoint == mountDir {
			if minfo.StartManifest != bzzHash ||
				minfo.LatestManifest != bzzHash ||
				minfo.fuseConnection == nil {
				t.Fatalf("Error mounting: exp(%s): act(%s)", bzzHash, minfo.StartManifest)
			}
			found = true
		}
		minfo.lock.RUnlock()
	}

	// Test listMounts
	if !found {
		t.Fatalf("Error getting mounts information for %v: %v", mountDir, err)
	}

	// Check if file and their attributes are as expected
	compareGeneratedFileWithFileInMount(t, files, mountDir)

	return swarmfs
}

func compareGeneratedFileWithFileInMount(t *testing.T, files map[string]fileInfo, mountDir string) {
	err := filepath.Walk(mountDir, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() {
			return nil
		}
		fname := path[len(mountDir)+1:]
		if _, ok := files[fname]; !ok {
			t.Fatalf(" file %v present in mount dir and is not expected", fname)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Error walking dir %v", mountDir)
	}

	for fname, finfo := range files {
		destinationFile := filepath.Join(mountDir, fname)

		dfinfo, err := os.Stat(destinationFile)
		if err != nil {
			t.Fatalf("Destination file %v missing in mount: %v", fname, err)
		}

		if int64(len(finfo.contents)) != dfinfo.Size() {
			t.Fatalf("file %v Size mismatch  source (%v) vs destination(%v)", fname, int64(len(finfo.contents)), dfinfo.Size())
		}

		if dfinfo.Mode().Perm().String() != "-rwx------" {
			t.Fatalf("file %v Permission mismatch source (-rwx------) vs destination(%v)", fname, dfinfo.Mode().Perm())
		}

		fileContents, err := ioutil.ReadFile(filepath.Join(mountDir, fname))
		if err != nil {
			t.Fatalf("Could not readfile %v : %v", fname, err)
		}
		if !bytes.Equal(fileContents, finfo.contents) {
			t.Fatalf("File %v contents mismatch: %v , %v", fname, fileContents, finfo.contents)

		}
		// TODO: check uid and gid
	}
}

func checkFile(t *testing.T, testMountDir, fname string, contents []byte) {
	destinationFile := filepath.Join(testMountDir, fname)
	dfinfo, err1 := os.Stat(destinationFile)
	if err1 != nil {
		t.Fatalf("Could not stat file %v", destinationFile)
	}
	if dfinfo.Size() != int64(len(contents)) {
		t.Fatalf("Mismatch in size  actual(%v) vs expected(%v)", dfinfo.Size(), int64(len(contents)))
	}

	fd, err2 := os.OpenFile(destinationFile, os.O_RDONLY, os.FileMode(0665))
	if err2 != nil {
		t.Fatalf("Could not open file %v", destinationFile)
	}
	newcontent := make([]byte, len(contents))
	fd.Read(newcontent)
	fd.Close()

	if !bytes.Equal(contents, newcontent) {
		t.Fatalf("File content mismatch expected (%v): received (%v) ", contents, newcontent)
	}
}

func getRandomBytes(size int) []byte {
	contents := make([]byte, size)
	rand.Read(contents)
	return contents
}

func isDirEmpty(name string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = f.Readdirnames(1)

	return err == io.EOF
}

type testAPI struct {
	api *api.Api
}

func (ta *testAPI) mountListAndUnmountEncrypted(t *testing.T) {
	ta.mountListAndUnmount(t, true)
}

func (ta *testAPI) mountListAndUnmountNonEncrypted(t *testing.T) {
	ta.mountListAndUnmount(t, false)
}

func (ta *testAPI) mountListAndUnmount(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "fuse-source")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "fuse-dest")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["2.txt"] = fileInfo{0711, 333, 444, getRandomBytes(10)}
	files["3.txt"] = fileInfo{0622, 333, 444, getRandomBytes(100)}
	files["4.txt"] = fileInfo{0533, 333, 444, getRandomBytes(1024)}
	files["5.txt"] = fileInfo{0544, 333, 444, getRandomBytes(10)}
	files["6.txt"] = fileInfo{0555, 333, 444, getRandomBytes(10)}
	files["7.txt"] = fileInfo{0666, 333, 444, getRandomBytes(10)}
	files["8.txt"] = fileInfo{0777, 333, 333, getRandomBytes(10)}
	files["11.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["111.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["two/2.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["two/2/2.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10)}
	files["two/2./2.txt"] = fileInfo{0777, 444, 444, getRandomBytes(10)}
	files["twice/2.txt"] = fileInfo{0777, 444, 333, getRandomBytes(200)}
	files["one/two/three/four/five/six/seven/eight/nine/10.txt"] = fileInfo{0777, 333, 444, getRandomBytes(10240)}
	files["one/two/three/four/five/six/six"] = fileInfo{0777, 333, 444, getRandomBytes(10)}

	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	// Check unmount
	_, err := swarmfs.Unmount(testMountDir)
	if err != nil {
		t.Fatalf("could not unmount  %v", bzzHash)
	}
	if !isDirEmpty(testMountDir) {
		t.Fatalf("unmount didnt work for %v", testMountDir)
	}
}

func (ta *testAPI) maxMountsEncrypted(t *testing.T) {
	ta.runMaxMounts(t, true)
}

func (ta *testAPI) maxMountsNonEncrypted(t *testing.T) {
	ta.runMaxMounts(t, false)
}

func (ta *testAPI) runMaxMounts(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir1, _ := ioutil.TempDir(os.TempDir(), "max-upload1")
	bzzHash1 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir1, toEncrypt)
	mount1, _ := ioutil.TempDir(os.TempDir(), "max-mount1")
	swarmfs1 := mountDir(t, ta.api, files, bzzHash1, mount1)
	defer swarmfs1.Stop()

	files["2.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir2, _ := ioutil.TempDir(os.TempDir(), "max-upload2")
	bzzHash2 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir2, toEncrypt)
	mount2, _ := ioutil.TempDir(os.TempDir(), "max-mount2")
	swarmfs2 := mountDir(t, ta.api, files, bzzHash2, mount2)
	defer swarmfs2.Stop()

	files["3.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir3, _ := ioutil.TempDir(os.TempDir(), "max-upload3")
	bzzHash3 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir3, toEncrypt)
	mount3, _ := ioutil.TempDir(os.TempDir(), "max-mount3")
	swarmfs3 := mountDir(t, ta.api, files, bzzHash3, mount3)
	defer swarmfs3.Stop()

	files["4.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir4, _ := ioutil.TempDir(os.TempDir(), "max-upload4")
	bzzHash4 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir4, toEncrypt)
	mount4, _ := ioutil.TempDir(os.TempDir(), "max-mount4")
	swarmfs4 := mountDir(t, ta.api, files, bzzHash4, mount4)
	defer swarmfs4.Stop()

	files["5.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir5, _ := ioutil.TempDir(os.TempDir(), "max-upload5")
	bzzHash5 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir5, toEncrypt)
	mount5, _ := ioutil.TempDir(os.TempDir(), "max-mount5")
	swarmfs5 := mountDir(t, ta.api, files, bzzHash5, mount5)
	defer swarmfs5.Stop()

	files["6.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir6, _ := ioutil.TempDir(os.TempDir(), "max-upload6")
	bzzHash6 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir6, toEncrypt)
	mount6, _ := ioutil.TempDir(os.TempDir(), "max-mount6")

	os.RemoveAll(mount6)
	os.MkdirAll(mount6, 0777)
	_, err := swarmfs.Mount(bzzHash6, mount6)
	if err == nil {
		t.Fatalf("Error: Going beyond max mounts  %v", bzzHash6)
	}
}

func (ta *testAPI) remountEncrypted(t *testing.T) {
	ta.remount(t, true)
}
func (ta *testAPI) remountNonEncrypted(t *testing.T) {
	ta.remount(t, false)
}

func (ta *testAPI) remount(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	uploadDir1, _ := ioutil.TempDir(os.TempDir(), "re-upload1")
	bzzHash1 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir1, toEncrypt)
	testMountDir1, _ := ioutil.TempDir(os.TempDir(), "re-mount1")
	swarmfs := mountDir(t, ta.api, files, bzzHash1, testMountDir1)
	defer swarmfs.Stop()

	uploadDir2, _ := ioutil.TempDir(os.TempDir(), "re-upload2")
	bzzHash2 := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir2, toEncrypt)
	testMountDir2, _ := ioutil.TempDir(os.TempDir(), "re-mount2")

	// try mounting the same hash second time
	os.RemoveAll(testMountDir2)
	os.MkdirAll(testMountDir2, 0777)
	_, err := swarmfs.Mount(bzzHash1, testMountDir2)
	if err != nil {
		t.Fatalf("Error mounting hash  %v", bzzHash1)
	}

	// mount a different hash in already mounted point
	_, err = swarmfs.Mount(bzzHash2, testMountDir1)
	if err == nil {
		t.Fatalf("Error mounting hash  %v", bzzHash2)
	}

	// mount nonexistent hash
	_, err = swarmfs.Mount("0xfea11223344", testMountDir1)
	if err == nil {
		t.Fatalf("Error mounting hash  %v", bzzHash2)
	}
}

func (ta *testAPI) unmountEncrypted(t *testing.T) {
	ta.unmount(t, true)
}

func (ta *testAPI) unmountNonEncrypted(t *testing.T) {
	ta.unmount(t, false)
}

func (ta *testAPI) unmount(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	uploadDir, _ := ioutil.TempDir(os.TempDir(), "ex-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "ex-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, uploadDir, toEncrypt)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	swarmfs.Unmount(testMountDir)

	mi := swarmfs.Listmounts()
	for _, minfo := range mi {
		if minfo.MountPoint == testMountDir {
			t.Fatalf("mount state not cleaned up in unmount case %v", testMountDir)
		}
	}
}

func (ta *testAPI) unmountWhenResourceBusyEncrypted(t *testing.T) {
	ta.unmountWhenResourceBusy(t, true)
}
func (ta *testAPI) unmountWhenResourceBusyNonEncrypted(t *testing.T) {
	ta.unmountWhenResourceBusy(t, false)
}

func (ta *testAPI) unmountWhenResourceBusy(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "ex-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "ex-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	actualPath := filepath.Join(testMountDir, "2.txt")
	d, err := os.OpenFile(actualPath, os.O_RDWR, os.FileMode(0700))
	d.Write(getRandomBytes(10))

	_, err = swarmfs.Unmount(testMountDir)
	if err != nil {
		t.Fatalf("could not unmount  %v", bzzHash)
	}
	d.Close()

	mi := swarmfs.Listmounts()
	for _, minfo := range mi {
		if minfo.MountPoint == testMountDir {
			t.Fatalf("mount state not cleaned up in unmount case %v", testMountDir)
		}
	}
}

func (ta *testAPI) seekInMultiChunkFileEncrypted(t *testing.T) {
	ta.seekInMultiChunkFile(t, true)
}

func (ta *testAPI) seekInMultiChunkFileNonEncrypted(t *testing.T) {
	ta.seekInMultiChunkFile(t, false)
}

func (ta *testAPI) seekInMultiChunkFile(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "seek-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "seek-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10240)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs.Stop()

	// Create a new file seek the second chunk
	actualPath := filepath.Join(testMountDir, "1.txt")
	d, _ := os.OpenFile(actualPath, os.O_RDONLY, os.FileMode(0700))

	d.Seek(5000, 0)

	contents := make([]byte, 1024)
	d.Read(contents)
	finfo := files["1.txt"]

	if !bytes.Equal(finfo.contents[:6024][5000:], contents) {
		t.Fatalf("File seek contents mismatch")
	}
	d.Close()
}

func (ta *testAPI) createNewFileEncrypted(t *testing.T) {
	ta.createNewFile(t, true)
}

func (ta *testAPI) createNewFileNonEncrypted(t *testing.T) {
	ta.createNewFile(t, false)
}

func (ta *testAPI) createNewFile(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "create-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "create-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file in the root dir and check
	actualPath := filepath.Join(testMountDir, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	contents := make([]byte, 11)
	rand.Read(contents)
	d.Write(contents)
	d.Close()

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}

	// mount again and see if things are okay
	files["2.txt"] = fileInfo{0700, 333, 444, contents}
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "2.txt", contents)
}

func (ta *testAPI) createNewFileInsideDirectoryEncrypted(t *testing.T) {
	ta.createNewFileInsideDirectory(t, true)
}

func (ta *testAPI) createNewFileInsideDirectoryNonEncrypted(t *testing.T) {
	ta.createNewFileInsideDirectory(t, false)
}

func (ta *testAPI) createNewFileInsideDirectory(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "createinsidedir-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "createinsidedir-mount")

	files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file inside a existing dir and check
	dirToCreate := filepath.Join(testMountDir, "one")
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	contents := make([]byte, 11)
	rand.Read(contents)
	d.Write(contents)
	d.Close()

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}

	// mount again and see if things are okay
	files["one/2.txt"] = fileInfo{0700, 333, 444, contents}
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "one/2.txt", contents)
}

func (ta *testAPI) createNewFileInsideNewDirectoryEncrypted(t *testing.T) {
	ta.createNewFileInsideNewDirectory(t, true)
}

func (ta *testAPI) createNewFileInsideNewDirectoryNonEncrypted(t *testing.T) {
	ta.createNewFileInsideNewDirectory(t, false)
}

func (ta *testAPI) createNewFileInsideNewDirectory(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "createinsidenewdir-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "createinsidenewdir-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file inside a existing dir and check
	dirToCreate := filepath.Join(testMountDir, "one")
	os.MkdirAll(dirToCreate, 0777)
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	contents := make([]byte, 11)
	rand.Read(contents)
	d.Write(contents)
	d.Close()

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}

	// mount again and see if things are okay
	files["one/2.txt"] = fileInfo{0700, 333, 444, contents}
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "one/2.txt", contents)
}

func (ta *testAPI) removeExistingFileEncrypted(t *testing.T) {
	ta.removeExistingFile(t, true)
}

func (ta *testAPI) removeExistingFileNonEncrypted(t *testing.T) {
	ta.removeExistingFile(t, false)
}

func (ta *testAPI) removeExistingFile(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "remove-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "remove-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Remove a file in the root dir and check
	actualPath := filepath.Join(testMountDir, "five.txt")
	os.Remove(actualPath)

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}

	// mount again and see if things are okay
	delete(files, "five.txt")
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) removeExistingFileInsideDirEncrypted(t *testing.T) {
	ta.removeExistingFileInsideDir(t, true)
}

func (ta *testAPI) removeExistingFileInsideDirNonEncrypted(t *testing.T) {
	ta.removeExistingFileInsideDir(t, false)
}

func (ta *testAPI) removeExistingFileInsideDir(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "remove-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "remove-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["one/five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["one/six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Remove a file in the root dir and check
	actualPath := filepath.Join(testMountDir, "one/five.txt")
	os.Remove(actualPath)

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}

	// mount again and see if things are okay
	delete(files, "one/five.txt")
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) removeNewlyAddedFileEncrypted(t *testing.T) {
	ta.removeNewlyAddedFile(t, true)
}

func (ta *testAPI) removeNewlyAddedFileNonEncrypted(t *testing.T) {
	ta.removeNewlyAddedFile(t, false)
}

func (ta *testAPI) removeNewlyAddedFile(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "removenew-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "removenew-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Adda a new file and remove it
	dirToCreate := filepath.Join(testMountDir, "one")
	os.MkdirAll(dirToCreate, os.FileMode(0665))
	actualPath := filepath.Join(dirToCreate, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	contents := make([]byte, 11)
	rand.Read(contents)
	d.Write(contents)
	d.Close()

	checkFile(t, testMountDir, "one/2.txt", contents)

	os.Remove(actualPath)

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}

	// mount again and see if things are okay
	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	if bzzHash != mi.LatestManifest {
		t.Fatalf("same contents different hash orig(%v): new(%v)", bzzHash, mi.LatestManifest)
	}
}

func (ta *testAPI) addNewFileAndModifyContentsEncrypted(t *testing.T) {
	ta.addNewFileAndModifyContents(t, true)
}

func (ta *testAPI) addNewFileAndModifyContentsNonEncrypted(t *testing.T) {
	ta.addNewFileAndModifyContents(t, false)
}

func (ta *testAPI) addNewFileAndModifyContents(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "modifyfile-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "modifyfile-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	// Create a new file in the root dir and check
	actualPath := filepath.Join(testMountDir, "2.txt")
	d, err1 := os.OpenFile(actualPath, os.O_RDWR|os.O_CREATE, os.FileMode(0665))
	if err1 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err1)
	}
	line1 := []byte("Line 1")
	rand.Read(line1)
	d.Write(line1)
	d.Close()

	mi1, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v", err2)
	}

	// mount again and see if things are okay
	files["2.txt"] = fileInfo{0700, 333, 444, line1}
	swarmfs2 := mountDir(t, ta.api, files, mi1.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "2.txt", line1)

	mi2, err3 := swarmfs2.Unmount(testMountDir)
	if err3 != nil {
		t.Fatalf("Could not unmount %v", err3)
	}

	// mount again and modify
	swarmfs3 := mountDir(t, ta.api, files, mi2.LatestManifest, testMountDir)
	defer swarmfs3.Stop()

	fd, err4 := os.OpenFile(actualPath, os.O_RDWR|os.O_APPEND, os.FileMode(0665))
	if err4 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err4)
	}
	line2 := []byte("Line 2")
	rand.Read(line2)
	fd.Seek(int64(len(line1)), 0)
	fd.Write(line2)
	fd.Close()

	mi3, err5 := swarmfs3.Unmount(testMountDir)
	if err5 != nil {
		t.Fatalf("Could not unmount %v", err5)
	}

	// mount again and see if things are okay
	b := [][]byte{line1, line2}
	line1and2 := bytes.Join(b, []byte(""))
	files["2.txt"] = fileInfo{0700, 333, 444, line1and2}
	swarmfs4 := mountDir(t, ta.api, files, mi3.LatestManifest, testMountDir)
	defer swarmfs4.Stop()

	checkFile(t, testMountDir, "2.txt", line1and2)
}

func (ta *testAPI) removeEmptyDirEncrypted(t *testing.T) {
	ta.removeEmptyDir(t, true)
}

func (ta *testAPI) removeEmptyDirNonEncrypted(t *testing.T) {
	ta.removeEmptyDir(t, false)
}

func (ta *testAPI) removeEmptyDir(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "rmdir-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "rmdir-mount")

	files["1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	os.MkdirAll(filepath.Join(testMountDir, "newdir"), 0777)

	mi, err3 := swarmfs1.Unmount(testMountDir)
	if err3 != nil {
		t.Fatalf("Could not unmount %v", err3)
	}
	if bzzHash != mi.LatestManifest {
		t.Fatalf("same contents different hash orig(%v): new(%v)", bzzHash, mi.LatestManifest)
	}
}

func (ta *testAPI) removeDirWhichHasFilesEncrypted(t *testing.T) {
	ta.removeDirWhichHasFiles(t, true)
}
func (ta *testAPI) removeDirWhichHasFilesNonEncrypted(t *testing.T) {
	ta.removeDirWhichHasFiles(t, false)
}

func (ta *testAPI) removeDirWhichHasFiles(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "rmdir-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "rmdir-mount")

	files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/five.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/six.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	dirPath := filepath.Join(testMountDir, "two")
	os.RemoveAll(dirPath)

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v ", err2)
	}

	// mount again and see if things are okay
	delete(files, "two/five.txt")
	delete(files, "two/six.txt")

	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) removeDirWhichHasSubDirsEncrypted(t *testing.T) {
	ta.removeDirWhichHasSubDirs(t, true)
}

func (ta *testAPI) removeDirWhichHasSubDirsNonEncrypted(t *testing.T) {
	ta.removeDirWhichHasSubDirs(t, false)
}
func (ta *testAPI) removeDirWhichHasSubDirs(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "rmsubdir-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "rmsubdir-mount")

	files["one/1.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/three/2.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/three/3.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/four/5.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/four/6.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}
	files["two/four/six/7.txt"] = fileInfo{0700, 333, 444, getRandomBytes(10)}

	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	dirPath := filepath.Join(testMountDir, "two")
	os.RemoveAll(dirPath)

	mi, err2 := swarmfs1.Unmount(testMountDir)
	if err2 != nil {
		t.Fatalf("Could not unmount %v ", err2)
	}

	// mount again and see if things are okay
	delete(files, "two/three/2.txt")
	delete(files, "two/three/3.txt")
	delete(files, "two/four/5.txt")
	delete(files, "two/four/6.txt")
	delete(files, "two/four/six/7.txt")

	swarmfs2 := mountDir(t, ta.api, files, mi.LatestManifest, testMountDir)
	defer swarmfs2.Stop()
}

func (ta *testAPI) appendFileContentsToEndEncrypted(t *testing.T) {
	ta.appendFileContentsToEnd(t, true)
}

func (ta *testAPI) appendFileContentsToEndNonEncrypted(t *testing.T) {
	ta.appendFileContentsToEnd(t, false)
}

func (ta *testAPI) appendFileContentsToEnd(t *testing.T, toEncrypt bool) {
	files := make(map[string]fileInfo)
	testUploadDir, _ := ioutil.TempDir(os.TempDir(), "appendlargefile-upload")
	testMountDir, _ := ioutil.TempDir(os.TempDir(), "appendlargefile-mount")

	line1 := make([]byte, 10)
	rand.Read(line1)
	files["1.txt"] = fileInfo{0700, 333, 444, line1}
	bzzHash := createTestFilesAndUploadToSwarm(t, ta.api, files, testUploadDir, toEncrypt)

	swarmfs1 := mountDir(t, ta.api, files, bzzHash, testMountDir)
	defer swarmfs1.Stop()

	actualPath := filepath.Join(testMountDir, "1.txt")
	fd, err4 := os.OpenFile(actualPath, os.O_RDWR|os.O_APPEND, os.FileMode(0665))
	if err4 != nil {
		t.Fatalf("Could not create file %s : %v", actualPath, err4)
	}
	line2 := make([]byte, 5)
	rand.Read(line2)
	fd.Seek(int64(len(line1)), 0)
	fd.Write(line2)
	fd.Close()

	mi1, err5 := swarmfs1.Unmount(testMountDir)
	if err5 != nil {
		t.Fatalf("Could not unmount %v ", err5)
	}

	// mount again and see if things are okay
	b := [][]byte{line1, line2}
	line1and2 := bytes.Join(b, []byte(""))
	files["1.txt"] = fileInfo{0700, 333, 444, line1and2}
	swarmfs2 := mountDir(t, ta.api, files, mi1.LatestManifest, testMountDir)
	defer swarmfs2.Stop()

	checkFile(t, testMountDir, "1.txt", line1and2)
}

func TestFUSE(t *testing.T) {
	datadir, err := ioutil.TempDir("", "fuse")
	if err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	os.RemoveAll(datadir)

	dpa, err := storage.NewLocalDPA(datadir, make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	ta := &testAPI{api: api.NewApi(dpa, nil, nil)}

	t.Run("mountListAndUnmountEncrypted", ta.mountListAndUnmountEncrypted)
	t.Run("mountListAndUnmountNonEncrypted", ta.mountListAndUnmountNonEncrypted)
	t.Run("maxMountsEncrypted", ta.maxMountsEncrypted)
	t.Run("maxMountsNonEncrypted", ta.maxMountsNonEncrypted)
	t.Run("remountEncrypted", ta.remountEncrypted)
	t.Run("remountNonEncrypted", ta.remountNonEncrypted)
	t.Run("unmountEncrypted", ta.unmountEncrypted)
	t.Run("unmountNonEncrypted", ta.unmountNonEncrypted)
	t.Run("unmountWhenResourceBusyEncrypted", ta.unmountWhenResourceBusyEncrypted)
	t.Run("unmountWhenResourceBusyNonEncrypted", ta.unmountWhenResourceBusyNonEncrypted)
	t.Run("seekInMultiChunkFileEncrypted", ta.seekInMultiChunkFileEncrypted)
	t.Run("seekInMultiChunkFileNonEncrypted", ta.seekInMultiChunkFileNonEncrypted)
	t.Run("createNewFileEncrypted", ta.createNewFileEncrypted)
	t.Run("createNewFileNonEncrypted", ta.createNewFileNonEncrypted)
	t.Run("createNewFileInsideDirectoryEncrypted", ta.createNewFileInsideDirectoryEncrypted)
	t.Run("createNewFileInsideDirectoryNonEncrypted", ta.createNewFileInsideDirectoryNonEncrypted)
	t.Run("createNewFileInsideNewDirectoryEncrypted", ta.createNewFileInsideNewDirectoryEncrypted)
	t.Run("createNewFileInsideNewDirectoryNonEncrypted", ta.createNewFileInsideNewDirectoryNonEncrypted)
	t.Run("removeExistingFileEncrypted", ta.removeExistingFileEncrypted)
	t.Run("removeExistingFileNonEncrypted", ta.removeExistingFileNonEncrypted)
	t.Run("removeExistingFileInsideDirEncrypted", ta.removeExistingFileInsideDirEncrypted)
	t.Run("removeExistingFileInsideDirNonEncrypted", ta.removeExistingFileInsideDirNonEncrypted)
	t.Run("removeNewlyAddedFileEncrypted", ta.removeNewlyAddedFileEncrypted)
	t.Run("removeNewlyAddedFileNonEncrypted", ta.removeNewlyAddedFileNonEncrypted)
	t.Run("addNewFileAndModifyContentsEncrypted", ta.addNewFileAndModifyContentsEncrypted)
	t.Run("addNewFileAndModifyContentsNonEncrypted", ta.addNewFileAndModifyContentsNonEncrypted)
	t.Run("removeEmptyDirEncrypted", ta.removeEmptyDirEncrypted)
	t.Run("removeEmptyDirNonEncrypted", ta.removeEmptyDirNonEncrypted)
	t.Run("removeDirWhichHasFilesEncrypted", ta.removeDirWhichHasFilesEncrypted)
	t.Run("removeDirWhichHasFilesNonEncrypted", ta.removeDirWhichHasFilesNonEncrypted)
	t.Run("removeDirWhichHasSubDirsEncrypted", ta.removeDirWhichHasSubDirsEncrypted)
	t.Run("removeDirWhichHasSubDirsNonEncrypted", ta.removeDirWhichHasSubDirsNonEncrypted)
	t.Run("appendFileContentsToEndEncrypted", ta.appendFileContentsToEndEncrypted)
	t.Run("appendFileContentsToEndNonEncrypted", ta.appendFileContentsToEndNonEncrypted)
}
