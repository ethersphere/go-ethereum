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
include CMakeFiles/teetransport.dir/depend.make

# Include the progress variables for this target.
include CMakeFiles/teetransport.dir/progress.make

# Include the compile flags for this target's objects.
include CMakeFiles/teetransport.dir/flags.make

CMakeFiles/teetransport.dir/teetransport/teetransport.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/teetransport.c.o: teetransport/teetransport.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_1)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/teetransport.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/teetransport.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/teetransport.c

CMakeFiles/teetransport.dir/teetransport/teetransport.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/teetransport.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/teetransport.c > CMakeFiles/teetransport.dir/teetransport/teetransport.c.i

CMakeFiles/teetransport.dir/teetransport/teetransport.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/teetransport.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/teetransport.c -o CMakeFiles/teetransport.dir/teetransport/teetransport.c.s

CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.requires

CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.provides: CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.provides

CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/teetransport.c.o

CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o: teetransport/teetransport_internal.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_2)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/teetransport_internal.c

CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/teetransport_internal.c > CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.i

CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/teetransport_internal.c -o CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.s

CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.requires

CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.provides: CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.provides

CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o: teetransport/transport/socket/teetransport_socket.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_3)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/teetransport_socket.c

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/teetransport_socket.c > CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.i

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/teetransport_socket.c -o CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.s

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o: teetransport/transport/socket/teetransport_socket_wrapper.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_4)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/teetransport_socket_wrapper.c

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/teetransport_socket_wrapper.c > CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.i

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/teetransport_socket_wrapper.c -o CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.s

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o

CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o: teetransport/transport/socket/lib/socket_linux.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_5)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/lib/socket_linux.c

CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/lib/socket_linux.c > CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.i

CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/socket/lib/socket_linux.c -o CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.s

CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o: teetransport/transport/libtee/teetransport_libtee.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_6)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee.c

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee.c > CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.i

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee.c -o CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.s

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o: teetransport/transport/libtee/teetransport_libtee_wrapper.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_7)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee_wrapper.c

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee_wrapper.c > CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.i

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee_wrapper.c -o CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.s

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o: teetransport/transport/libtee/teetransport_libtee_client_metadata.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_8)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee_client_metadata.c

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee_client_metadata.c > CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.i

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/libtee/teetransport_libtee_client_metadata.c -o CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.s

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o: teetransport/transport/dal_device/teetransport_dal_device.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_9)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/dal_device/teetransport_dal_device.c

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/dal_device/teetransport_dal_device.c > CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.i

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/dal_device/teetransport_dal_device.c -o CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.s

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o: teetransport/transport/dal_device/teetransport_dal_device_wrapper.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_10)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c > CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.i

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c -o CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.s

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.requires

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.provides: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.provides

CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.provides.build: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o: thirdparty/libtee/linux/libteelinux.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_11)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/thirdparty/libtee/linux/libteelinux.c

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/thirdparty/libtee/linux/libteelinux.c > CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.i

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/thirdparty/libtee/linux/libteelinux.c -o CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.s

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.requires

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.provides: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.provides

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.provides.build: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o: CMakeFiles/teetransport.dir/flags.make
CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o: thirdparty/libtee/linux/libmei/mei.c
	$(CMAKE_COMMAND) -E cmake_progress_report /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles $(CMAKE_PROGRESS_12)
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Building C object CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -o CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o   -c /var/www/vhost/sgx/dynamic-application-loader-host-interface/thirdparty/libtee/linux/libmei/mei.c

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.i: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Preprocessing C source to CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.i"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -E /var/www/vhost/sgx/dynamic-application-loader-host-interface/thirdparty/libtee/linux/libmei/mei.c > CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.i

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.s: cmake_force
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --green "Compiling C source to assembly CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.s"
	/usr/bin/cc  $(C_DEFINES) $(C_FLAGS) -S /var/www/vhost/sgx/dynamic-application-loader-host-interface/thirdparty/libtee/linux/libmei/mei.c -o CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.s

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.requires:
.PHONY : CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.requires

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.provides: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.requires
	$(MAKE) -f CMakeFiles/teetransport.dir/build.make CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.provides.build
.PHONY : CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.provides

CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.provides.build: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o

# Object files for target teetransport
teetransport_OBJECTS = \
"CMakeFiles/teetransport.dir/teetransport/teetransport.c.o" \
"CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o" \
"CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o" \
"CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o" \
"CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o"

# External object files for target teetransport
teetransport_EXTERNAL_OBJECTS =

bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/teetransport.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/build.make
bin_linux/libteetransport.so: CMakeFiles/teetransport.dir/link.txt
	@$(CMAKE_COMMAND) -E cmake_echo_color --switch=$(COLOR) --red --bold "Linking C shared library bin_linux/libteetransport.so"
	$(CMAKE_COMMAND) -E cmake_link_script CMakeFiles/teetransport.dir/link.txt --verbose=$(VERBOSE)

# Rule to build all files generated by this target.
CMakeFiles/teetransport.dir/build: bin_linux/libteetransport.so
.PHONY : CMakeFiles/teetransport.dir/build

CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/teetransport.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/teetransport_internal.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/socket/teetransport_socket_wrapper.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/socket/lib/socket_linux.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_wrapper.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/libtee/teetransport_libtee_client_metadata.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/teetransport/transport/dal_device/teetransport_dal_device_wrapper.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libteelinux.c.o.requires
CMakeFiles/teetransport.dir/requires: CMakeFiles/teetransport.dir/thirdparty/libtee/linux/libmei/mei.c.o.requires
.PHONY : CMakeFiles/teetransport.dir/requires

CMakeFiles/teetransport.dir/clean:
	$(CMAKE_COMMAND) -P CMakeFiles/teetransport.dir/cmake_clean.cmake
.PHONY : CMakeFiles/teetransport.dir/clean

CMakeFiles/teetransport.dir/depend:
	cd /var/www/vhost/sgx/dynamic-application-loader-host-interface && $(CMAKE_COMMAND) -E cmake_depends "Unix Makefiles" /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface /var/www/vhost/sgx/dynamic-application-loader-host-interface/CMakeFiles/teetransport.dir/DependInfo.cmake --color=$(COLOR)
.PHONY : CMakeFiles/teetransport.dir/depend

