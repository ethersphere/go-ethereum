/*
 * Copyright (C) 2011-2017 Intel Corporation. All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 *
 *   * Redistributions of source code must retain the above copyright
 *     notice, this list of conditions and the following disclaimer.
 *   * Redistributions in binary form must reproduce the above copyright
 *     notice, this list of conditions and the following disclaimer in
 *     the documentation and/or other materials provided with the
 *     distribution.
 *   * Neither the name of Intel Corporation nor the names of its
 *     contributors may be used to endorse or promote products derived
 *     from this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 *
 */



#ifndef _SGX_DH_H_
#define _SGX_DH_H_

#include "sgx.h"
#include "sgx_defs.h"
#include "sgx_ecp_types.h"

#pragma pack(push, 1)

#define SGX_DH_MAC_SIZE 16

#define SGX_DH_SESSION_DATA_SIZE 200

typedef struct _sgx_dh_msg1_t
{
    sgx_ec256_public_t  g_a;     /* the Endian-ness of Ga is Little-Endian */
    sgx_target_info_t   target;
} sgx_dh_msg1_t;

typedef struct _sgx_dh_msg2_t
{
    sgx_ec256_public_t  g_b;     /* the Endian-ness of Gb is Little-Endian */
    sgx_report_t        report;
    uint8_t             cmac[SGX_DH_MAC_SIZE];
} sgx_dh_msg2_t;

typedef struct _sgx_dh_msg3_body_t
{
    sgx_report_t report;
    uint32_t     additional_prop_length;
    uint8_t      additional_prop[0];
} sgx_dh_msg3_body_t;


typedef struct _sgx_dh_msg3_t
{
    uint8_t            cmac[SGX_DH_MAC_SIZE];
    sgx_dh_msg3_body_t msg3_body;
} sgx_dh_msg3_t;

typedef struct _sgx_dh_session_enclave_identity_t
{
    sgx_cpu_svn_t     cpu_svn;
    sgx_misc_select_t misc_select;
    uint8_t           reserved_1[28];
    sgx_attributes_t  attributes;
    sgx_measurement_t mr_enclave;
    uint8_t           reserved_2[32];
    sgx_measurement_t mr_signer;
    uint8_t           reserved_3[96];
    sgx_prod_id_t     isv_prod_id;
    sgx_isv_svn_t     isv_svn;
} sgx_dh_session_enclave_identity_t;

typedef enum _sgx_dh_session_role_t
{
    SGX_DH_SESSION_INITIATOR,
    SGX_DH_SESSION_RESPONDER
} sgx_dh_session_role_t;

typedef struct _sgx_dh_session_t
{
    uint8_t sgx_dh_session[SGX_DH_SESSION_DATA_SIZE];
} sgx_dh_session_t;
#pragma pack(pop)
#ifdef __cplusplus
extern "C" {
#endif

/* The order of calling SGX DH Library APIs is restricted as below */
/* As session initiator : Step.1 sgx_dh_init_session -->  Step.2 sgx_dh_initiator_proc_msg1 --> Step.3 sgx_dh_initiator_proc_msg3 */
/* As session responder :  Step.1 sgx_dh_init_session --> Step.2 sgx_dh_responder_gen_msg1 --> Step.3 sgx_dh_responder_proc_msg2*/
/* Any out of order calling will cause session establishment failure. */

/*Function name: sgx_dh_init_session
** parameter description
**@ [input] role: caller's role in dh session establishment
**@ [output] session: point to dh session structure that is used during establishment, the buffer must be in enclave address space
*/
sgx_status_t SGXAPI sgx_dh_init_session(sgx_dh_session_role_t role,
                                        sgx_dh_session_t* session);
/*Function name: sgx_dh_responder_gen_msg1
** parameter description
**@ [output] msg1: point to dh message 1 buffer, and the buffer must be in enclave address space
**@ [input/output] dh_session: point to dh session structure that is used during establishment, and the buffer must be in enclave address space
*/
sgx_status_t SGXAPI sgx_dh_responder_gen_msg1(sgx_dh_msg1_t* msg1,
                                              sgx_dh_session_t* dh_session);
/*Function name: sgx_dh_initiator_proc_msg1
** parameter description
**@ [input] msg1: point to dh message 1 buffer generated by session responder, and the buffer must be in enclave address space
**@ [output] msg2: point to dh message 2 buffer, and the buffer must be in enclave address space
**@ [input/output] dh_session: point to dh session structure that is used during establishment, and the buffer must be in enclave address space
*/
sgx_status_t SGXAPI sgx_dh_initiator_proc_msg1(const sgx_dh_msg1_t* msg1,
                                               sgx_dh_msg2_t* msg2,
                                               sgx_dh_session_t* dh_session);
/*Function name: sgx_dh_responder_proc_msg2
** parameter description
**@ [input] msg2: point to dh message 2 buffer generated by session initiator, and the buffer must be in enclave address space
**@ [output] msg3: point to dh message 3 buffer generated by session responder in this function, and the buffer must be in enclave address space
**@ [input/output] dh_session: point to dh session structure that is used during establishment, and the buffer must be in enclave address space
**@ [output] aek: AEK derived from shared key. the buffer must be in enclave address space.
**@ [output] initiator_identity: identity information of initiator including isv svn, isv product id, sgx attributes, mr signer, and mr enclave. the buffer must be in enclave address space.
*/
sgx_status_t SGXAPI sgx_dh_responder_proc_msg2(const sgx_dh_msg2_t* msg2,
                                               sgx_dh_msg3_t* msg3,
                                               sgx_dh_session_t* dh_session,
                                               sgx_key_128bit_t* aek,
                                               sgx_dh_session_enclave_identity_t* initiator_identity);
/*Function name: sgx_dh_initiator_proc_msg3
** parameter description
**@ [input] msg3: point to dh message 3 buffer generated by session responder, and the buffer must be in enclave address space
**@ [input/output] dh_session: point to dh session structure that is used during establishment, and the buffer must be in enclave address space
**@ [output] aek: AEK derived from shared key. the buffer must be in enclave address space.
**@ [output] responder_identity: identity information of responder including isv svn, isv product id, sgx attributes, mr signer, and mr enclave. the buffer must be in enclave address space.
*/
sgx_status_t SGXAPI sgx_dh_initiator_proc_msg3(const sgx_dh_msg3_t* msg3,
                                               sgx_dh_session_t* dh_session,
                                               sgx_key_128bit_t* aek,
                                               sgx_dh_session_enclave_identity_t* responder_identity);

#ifdef __cplusplus
}
#endif


#endif
