# CMAKE generated file: DO NOT EDIT!
# Generated by "Unix Makefiles" Generator, CMake Version 2.8

#=============================================================================
# Special targets provided by cmake.

# Disable implicit rules so canonical targets will work.
.SUFFIXES:

# Remove some rules from gmake that .SUFFIXES does not remove.
SUFFIXES =

.SUFFIXES: .hpux_make_needs_suffix_list

# Suppress display of executed commands.
$(VERBOSE).SILENT:

# A target that is always out of date.
cmake_force:
.PHONY : cmake_force

#=============================================================================
# Set environment variables for the build.

# The shell in which to execute make rules.
SHELL = /bin/sh

# The CMake executable.
CMAKE_COMMAND = /usr/bin/cmake

# The command to remove a file.
RM = /usr/bin/cmake -E remove -f

# Escaping for special characters.
EQUALS = =

# The program to use to edit the cache.
CMAKE_EDIT_COMMAND = /usr/bin/ccmake

# The top-level source directory on which CMake was run.
CMAKE_SOURCE_DIR = /var/www/vhost/sgx/dynamic-application-loader-host-interface

# The top-level build directory on which CMake was run.
CMAKE_BINARY_DIR = /var/www/vhost/sgx/dynamic-application-loader-host-interface

# Include any dependencies generated for this target.
include CMakeFiles/smoketest.dir/depend.make

# Include the progress variables for this target.
include CMakeFiles/smoketest.dir/progress.make

# Include the compile flags for this target's objects.
include CMakeFiles/smoketest.dir/flags.make

CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o: CMakeFiles/smoketest.dir/flags.make
CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o: test/smoketest/smoketest.cpp
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_1)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building CXX object CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o"
	/usr/bin/c++   $(CXX_DEFINES) $(CXX_FLAGS) -o CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/test/smoketest/smoketest.cpp

CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing CXX source to CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.i"
	/usr/bin/c++  $(CXX_DEFINES) $(CXX_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/test/smoketest/smoketest.cpp > CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.i

CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling CXX source to assembly CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.s"
	/usr/bin/c++  $(CXX_DEFINES) $(CXX_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/test/smoketest/smoketest.cpp -o CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.s

CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.requires:
.PHONY : CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.requires

CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.provides: CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.requires
	$(MAKE) -f CMakeFiles/smoketest.dir/build.make CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.provides.build
.PHONY : CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.provides

CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.provides.build: CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o

CMakeFiles/smoketest.dir/common/reg-linux.cpp.o: CMakeFiles/smoketest.dir/flags.make
CMakeFiles/smoketest.dir/common/reg-linux.cpp.o: common/reg-linux.cpp
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_2)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building CXX object CMakeFiles/smoketest.dir/common/reg-linux.cpp.o"
	/usr/bin/c++   $(CXX_DEFINES) $(CXX_FLAGS) -o CMakeFiles/smoketest.dir/common/reg-linux.cpp.o -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/common/reg-linux.cpp

CMakeFiles/smoketest.dir/common/reg-linux.cpp.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing CXX source to CMakeFiles/smoketest.dir/common/reg-linux.cpp.i"
	/usr/bin/c++  $(CXX_DEFINES) $(CXX_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/common/reg-linux.cpp > CMakeFiles/smoketest.dir/common/reg-linux.cpp.i

CMakeFiles/smoketest.dir/common/reg-linux.cpp.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling CXX source to assembly CMakeFiles/smoketest.dir/common/reg-linux.cpp.s"
	/usr/bin/c++  $(CXX_DEFINES) $(CXX_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/common/reg-linux.cpp -o CMakeFiles/smoketest.dir/common/reg-linux.cpp.s

CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.requires:
.PHONY : CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.requires

CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.provides: CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.requires
	$(MAKE) -f CMakeFiles/smoketest.dir/build.make CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.provides.build
.PHONY : CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.provides

CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.provides.build: CMakeFiles/smoketest.dir/common/reg-linux.cpp.o

CMakeFiles/smoketest.dir/common/misc.cpp.o: CMakeFiles/smoketest.dir/flags.make
CMakeFiles/smoketest.dir/common/misc.cpp.o: common/misc.cpp
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_3)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building CXX object CMakeFiles/smoketest.dir/common/misc.cpp.o"
	/usr/bin/c++   $(CXX_DEFINES) $(CXX_FLAGS) -o CMakeFiles/smoketest.dir/common/misc.cpp.o -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/common/misc.cpp

CMakeFiles/smoketest.dir/common/misc.cpp.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing CXX source to CMakeFiles/smoketest.dir/common/misc.cpp.i"
	/usr/bin/c++  $(CXX_DEFINES) $(CXX_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/common/misc.cpp > CMakeFiles/smoketest.dir/common/misc.cpp.i

CMakeFiles/smoketest.dir/common/misc.cpp.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling CXX source to assembly CMakeFiles/smoketest.dir/common/misc.cpp.s"
	/usr/bin/c++  $(CXX_DEFINES) $(CXX_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/common/misc.cpp -o CMakeFiles/smoketest.dir/common/misc.cpp.s

CMakeFiles/smoketest.dir/common/misc.cpp.o.requires:
.PHONY : CMakeFiles/smoketest.dir/common/misc.cpp.o.requires

CMakeFiles/smoketest.dir/common/misc.cpp.o.provides: CMakeFiles/smoketest.dir/common/misc.cpp.o.requires
	$(MAKE) -f CMakeFiles/smoketest.dir/build.make CMakeFiles/smoketest.dir/common/misc.cpp.o.provides.build
.PHONY : CMakeFiles/smoketest.dir/common/misc.cpp.o.provides

CMakeFiles/smoketest.dir/common/misc.cpp.o.provides.build: CMakeFiles/smoketest.dir/common/misc.cpp.o

# Object files for target smoketest
smoketest_OBJECTS = \
"CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o" \
"CMakeFiles/smoketest.dir/common/reg-linux.cpp.o" \
"CMakeFiles/smoketest.dir/common/misc.cpp.o"

# External object files for target smoketest
smoketest_EXTERNAL_OBJECTS =

bin_linux/smoketest: CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o
bin_linux/smoketest: CMakeFiles/smoketest.dir/common/reg-linux.cpp.o
bin_linux/smoketest: CMakeFiles/smoketest.dir/common/misc.cpp.o
bin_linux/smoketest: CMakeFiles/smoketest.dir/build.make
bin_linux/smoketest: bin_linux/libjhi.so
bin_linux/smoketest: bin_linux/libteemanagement.so
bin_linux/smoketest: bin_linux/libjhi.so
bin_linux/smoketest: CMakeFiles/smoketest.dir/link.txt
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --red --bold "Linking CXX executable bin_linux/smoketest"
	$(CMAKE_COMMAND) -E cmake_link_script CMakeFiles/smoketest.dir/link.txt --verbose=$(VERBOSE)

# Rule to build all files generated by this target.
CMakeFiles/smoketest.dir/build: bin_linux/smoketest
.PHONY : CMakeFiles/smoketest.dir/build

CMakeFiles/smoketest.dir/requires: CMakeFiles/smoketest.dir/test/smoketest/smoketest.cpp.o.requires
CMakeFiles/smoketest.dir/requires: CMakeFiles/smoketest.dir/common/reg-linux.cpp.o.requires
CMakeFiles/smoketest.dir/requires: CMakeFiles/smoketest.dir/common/misc.cpp.o.requires
.PHONY : CMakeFiles/smoketest.dir/requires

CMakeFiles/smoketest.dir/clean:
	$(CMAKE_COMMAND) -P CMakeFiles/smoketest.dir/cmake_clean.cmake
.PHONY : CMakeFiles/smoketest.dir/clean

CMakeFiles/smoketest.dir/depend:
	cd /var/www/vhost/sgx/dynamic-application-loader-host-interface && $(CMAKE_COMMAND) -E cmake_depends "Unix Makefiles" /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles/smoketest.dir/DependInfo.cmake --color=$(COLOR)
.PHONY : CMakeFiles/smoketest.dir/depend

