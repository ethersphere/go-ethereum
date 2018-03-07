/*############################################################################
  # Copyright 2016 Intel Corporation
  #
  # Licensed under the Apache License, Version 2.0 (the "License");
  # you may not use this file except in compliance with the License.
  # You may obtain a copy of the License at
  #
  #     http://www.apache.org/licenses/LICENSE-2.0
  #
  # Unless required by applicable law or agreed to in writing, software
  # distributed under the License is distributed on an "AS IS" BASIS,
  # WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  # See the License for the specific language governing permissions and
  # limitations under the License.
  ############################################################################*/

/*!
 * \file
 * \brief Argument parsing utilities interface.
 */
#ifndef EXAMPLE_UTIL_ARGUTIL_H_
#define EXAMPLE_UTIL_ARGUTIL_H_

/// get the index of an option in argv
/*!
  \param[in] argc number of arguments
  \param[in] argv list of arguments
  \param[in] name of option
  \returns index of the option in argv
*/
int GetOptionIndex(int argc, char* const argv[], char const* option);

/// test if an option is in argv
/*!
  \param[in] argc number of arguments
  \param[in] argv list of arguments
  \param[in] name of option
  \retval true option is in argv
  \retval false option is not in argv
*/
int CmdOptionExists(int argc, char* const argv[], char const* option);

/// find option in argv
/*!
  \param[in] argc number of arguments
  \param[in] argv list of arguments
  \param[in] name of option
  \returns pointer from argv for option
*/
char const* GetCmdOption(int argc, char* const argv[], char const* option);

#endif  // EXAMPLE_UTIL_ARGUTIL_H_
