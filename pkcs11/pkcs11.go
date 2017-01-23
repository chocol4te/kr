package main

/*
#include <stdio.h>
#include <stdlib.h>
#include "pkcs11.h"

#include <dlfcn.h>

static int dlopen_kr_logging_module() {
	void *handle;
	void (*Init)(void);
	char *error;

	handle = dlopen ("libkrlogging.dylib", RTLD_LAZY);
	if (!handle) {
		return 1;
	}

	Init = dlsym(handle, "Init");
	if ((error = dlerror()) != NULL)  {
		fputs(error, stderr);
		exit(1);
	}

	Init();
	return 0;
}
*/
import (
	"C"
)

import (
	"os"
	"sync"
	"unsafe"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krdclient"
	"github.com/op/go-logging"
)

type CK_SESSION_HANDLE C.CK_SESSION_HANDLE
type CK_SESSION_HANDLE_PTR C.CK_SESSION_HANDLE_PTR
type CK_OBJECT_HANDLE C.CK_OBJECT_HANDLE
type CK_ATTRIBUTE C.CK_ATTRIBUTE

const CKR_OK = C.CKR_OK
const CKF_SERIAL_SESSION = C.CKF_SERIAL_SESSION
const CKA_CLASS = C.CKA_CLASS
const CKO_PUBLIC_KEY = C.CKO_PUBLIC_KEY
const CKO_PRIVATE_KEY = C.CKO_PRIVATE_KEY

type ULONG C.CK_ULONG

var log = kr.SetupLogging("", logging.WARNING, os.Getenv("KR_LOG_SYSLOG") != "")

var mutex sync.Mutex

//export C_GetFunctionList
func C_GetFunctionList(l **C.CK_FUNCTION_LIST) C.CK_RV {

	log.Notice("GetFunctionList")
	*l = &functions
	return C.CKR_OK
}

//export C_Initialize
func C_Initialize(C.CK_VOID_PTR) C.CK_RV {
	log.Notice("Initialize")
	C.dlopen_kr_logging_module()

	mutex.Lock()
	defer mutex.Unlock()
	if !checkedForUpdate {
		CheckForUpdate()
		checkedForUpdate = true
	}

	return C.CKR_OK
}

//export C_GetInfo
func C_GetInfo(ck_info *C.CK_INFO) C.CK_RV {
	log.Notice("GetInfo")
	*ck_info = C.CK_INFO{
		cryptokiVersion: C.struct__CK_VERSION{
			major: 2,
			minor: 20,
		},
		flags:              0,
		manufacturerID:     bytesToChar32([]byte("KryptCo Inc.")),
		libraryDescription: bytesToChar32([]byte("Kryptonite pkcs11 middleware")),
		libraryVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
	}
	return C.CKR_OK
}

//export C_GetSlotList
func C_GetSlotList(token_present C.uchar, slot_list *C.CK_SLOT_ID, count *C.ulong) C.CK_RV {
	log.Notice("GetSlotList input count", *count)
	if slot_list == nil {
		log.Notice("slot_list nil")
		//	just return count
		*count = 1
		return C.CKR_OK
	}
	if *count == 0 {
		log.Notice("buffer too small")
		return C.CKR_BUFFER_TOO_SMALL
	}
	*count = 1
	*slot_list = 0
	return C.CKR_OK
}

//export C_GetSlotInfo
func C_GetSlotInfo(slotID C.CK_SLOT_ID, slotInfo *C.CK_SLOT_INFO) C.CK_RV {
	log.Notice("GetSlotInfo")
	*slotInfo = C.CK_SLOT_INFO{
		manufacturerID:  bytesToChar32([]byte("KryptCo, Inc.")),
		slotDescription: bytesToChar64([]byte("Kryptonite pkcs11 middleware")),
		hardwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		firmwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		flags: C.CKF_TOKEN_PRESENT | C.CKF_REMOVABLE_DEVICE,
	}

	return C.CKR_OK
}

//export C_GetTokenInfo
func C_GetTokenInfo(slotID C.CK_SLOT_ID, tokenInfo *C.CK_TOKEN_INFO) C.CK_RV {
	log.Notice("GetTokenInfo")
	*tokenInfo = C.CK_TOKEN_INFO{
		label:               bytesToChar32([]byte("Kryptonite iOS")),
		manufacturerID:      bytesToChar32([]byte("KryptCo Inc.")),
		model:               bytesToChar16([]byte("Kryptonite iOS")),
		serialNumber:        bytesToChar16([]byte("1")),
		ulMaxSessionCount:   16,
		ulSessionCount:      0,
		ulMaxRwSessionCount: 16,
		ulRwSessionCount:    0,
		ulMaxPinLen:         0,
		ulMinPinLen:         0,
		hardwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		firmwareVersion: C.struct__CK_VERSION{
			major: 0,
			minor: 1,
		},
		flags: C.CKF_TOKEN_INITIALIZED,
	}
	return C.CKR_OK
}

var nextSessionIota = CK_SESSION_HANDLE(1)

//export C_OpenSession
func C_OpenSession(slotID C.CK_SLOT_ID, flags C.CK_FLAGS, pApplication C.CK_VOID_PTR,
	notify C.CK_NOTIFY, sessionHandle *CK_SESSION_HANDLE) C.CK_RV {
	mutex.Lock()
	defer mutex.Unlock()
	log.Notice("OpenSession")
	if flags&C.CKF_SERIAL_SESSION == 0 {
		log.Error("CKF_SERIAL_SESSION not set")
		return C.CKR_SESSION_PARALLEL_NOT_SUPPORTED
	}
	if notify != nil {
		log.Warning("notify callback passed in, but not supported")
	}
	*sessionHandle = nextSessionIota
	nextSessionIota++
	return C.CKR_OK
}

//export C_GetSessionInfo
func C_GetSessionInfo(session CK_SESSION_HANDLE, info *C.CK_SESSION_INFO) C.CK_RV {
	log.Notice("GetSessionInfo")
	*info = C.CK_SESSION_INFO{
		slotID: 0,
		state:  C.CKS_RW_USER_FUNCTIONS,
		flags:  C.CKF_RW_SESSION | C.CKF_SERIAL_SESSION,
	}
	return C.CKR_OK
}

//export C_GetMechanismList
func C_GetMechanismList(slotID C.CK_SLOT_ID, mechList *C.CK_MECHANISM_TYPE, count *C.CK_ULONG) C.CK_RV {
	log.Notice("GetMechanismList")
	*count = C.CK_ULONG(0)
	return C.CKR_OK
}

//export C_GetMechanismInfo
func C_GetMechanismInfo(slotID C.CK_SLOT_ID, _type C.CK_MECHANISM_TYPE, info *C.CK_MECHANISM_INFO) C.CK_RV {
	log.Notice("GetMechanismInfo")
	if _type == C.CKM_RSA_PKCS {
		log.Notice("CKM_RSA_PKCS")
		*info = C.CK_MECHANISM_INFO{
			ulMinKeySize: 4096,
			ulMaxKeySize: 4096,
			flags:        C.CKF_SIGN | C.CKF_HW,
		}
	}
	return C.CKR_OK
}

//export C_CloseSession
func C_CloseSession(session CK_SESSION_HANDLE) C.CK_RV {
	log.Notice("CloseSession")
	mutex.Lock()
	defer mutex.Unlock()
	return C.CKR_OK
}

var sessionFoundObjects map[CK_SESSION_HANDLE]map[CK_OBJECT_HANDLE]bool = map[CK_SESSION_HANDLE]map[CK_OBJECT_HANDLE]bool{}
var sessionFindingObjects map[CK_SESSION_HANDLE]map[CK_OBJECT_HANDLE]bool = map[CK_SESSION_HANDLE]map[CK_OBJECT_HANDLE]bool{}

//export C_FindObjectsInit
func C_FindObjectsInit(session CK_SESSION_HANDLE, templates *CK_ATTRIBUTE, count ULONG) C.CK_RV {
	log.Notice("FindObjectsInit")
	mutex.Lock()
	defer mutex.Unlock()
	if count == 0 {
		log.Notice("count == 0, find all objects")
		return C.CKR_OK
	}
	for i := ULONG(0); i < count; i++ {
		log.Notice("Template type:", templates._type)
		switch templates._type {
		case C.CKA_CLASS:
			switch *(*C.CK_OBJECT_CLASS)(templates.pValue) {
			case C.CKO_PUBLIC_KEY:
				log.Notice("init search for CKO_PUBLIC_KEY")
				go krdclient.RequestNoOp()
			case C.CKO_PRIVATE_KEY:
				log.Notice("init search for CKO_PRIVATE_KEY")
			case C.CKO_MECHANISM:
				log.Notice("init search for CKO_MECHANISM unsupported")
			case C.CKO_CERTIFICATE:
				log.Notice("init search for CKO_CERTIFICATE unsupported")
			}
		}
		templates = (*CK_ATTRIBUTE)(unsafe.Pointer(uintptr(unsafe.Pointer(templates)) + unsafe.Sizeof(*templates)))
	}
	return C.CKR_OK
}

//export C_FindObjects
func C_FindObjects(session CK_SESSION_HANDLE, objects *CK_OBJECT_HANDLE, maxCount ULONG, count *ULONG) C.CK_RV {
	log.Notice("FindObjects")
	*count = 0
	return C.CKR_OK
}

//export C_FindObjectsFinal
func C_FindObjectsFinal(session CK_SESSION_HANDLE) C.CK_RV {
	return C.CKR_OK
}

var checkedForUpdate = false

//export C_GetAttributeValue
func C_GetAttributeValue(session CK_SESSION_HANDLE, object CK_OBJECT_HANDLE, template *CK_ATTRIBUTE, count C.CK_ULONG) C.CK_RV {
	mutex.Lock()
	defer mutex.Unlock()
	log.Notice("C_GetAttributeValue")

	return C.CKR_OK
}

//export C_SignInit
func C_SignInit(session CK_SESSION_HANDLE, mechanism C.CK_MECHANISM_PTR, key CK_OBJECT_HANDLE) C.CK_RV {
	log.Notice("C_SignInit mechanism", mechanism.mechanism)
	switch mechanism.mechanism {
	case C.CKM_RSA_PKCS:
		return C.CKR_OK
	case C.CKM_RSA_X_509:
		log.Error("CKM_RSA_X_509 not supported")
		return C.CKR_MECHANISM_INVALID
	default:
		return C.CKR_MECHANISM_INVALID
	}
	return C.CKR_OK
}

//export C_Sign
func C_Sign(session CK_SESSION_HANDLE,
	data C.CK_BYTE_PTR, dataLen C.ulong,
	signature C.CK_BYTE_PTR, signatureLen *C.ulong) C.CK_RV {
	log.Notice("C_Sign")
	log.Notice("input signatureLen", *signatureLen, "dataLen", dataLen)
	return C.CKR_OK
}

//export C_Finalize
func C_Finalize(reserved C.CK_VOID_PTR) C.CK_RV {
	log.Notice("Finalize")
	return C.CKR_OK
}

func bytesToChar64(b []byte) [64]C.uchar {
	for len(b) < 64 {
		b = append(b, byte(0))
	}
	return [64]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
		C.uchar(b[16]), C.uchar(b[17]), C.uchar(b[18]), C.uchar(b[19]),
		C.uchar(b[20]), C.uchar(b[21]), C.uchar(b[22]), C.uchar(b[23]),
		C.uchar(b[24]), C.uchar(b[25]), C.uchar(b[26]), C.uchar(b[27]),
		C.uchar(b[28]), C.uchar(b[29]), C.uchar(b[30]), C.uchar(b[31]),
		C.uchar(b[32]), C.uchar(b[33]), C.uchar(b[34]), C.uchar(b[35]),
		C.uchar(b[36]), C.uchar(b[37]), C.uchar(b[38]), C.uchar(b[39]),
		C.uchar(b[40]), C.uchar(b[41]), C.uchar(b[42]), C.uchar(b[43]),
		C.uchar(b[44]), C.uchar(b[45]), C.uchar(b[46]), C.uchar(b[47]),
		C.uchar(b[48]), C.uchar(b[49]), C.uchar(b[50]), C.uchar(b[51]),
		C.uchar(b[52]), C.uchar(b[53]), C.uchar(b[54]), C.uchar(b[55]),
		C.uchar(b[56]), C.uchar(b[57]), C.uchar(b[58]), C.uchar(b[59]),
		C.uchar(b[60]), C.uchar(b[61]), C.uchar(b[62]), C.uchar(b[63]),
	}
}

func bytesToChar32(b []byte) [32]C.uchar {
	for len(b) < 32 {
		b = append(b, byte(0))
	}
	return [32]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
		C.uchar(b[16]), C.uchar(b[17]), C.uchar(b[18]), C.uchar(b[19]),
		C.uchar(b[20]), C.uchar(b[21]), C.uchar(b[22]), C.uchar(b[23]),
		C.uchar(b[24]), C.uchar(b[25]), C.uchar(b[26]), C.uchar(b[27]),
		C.uchar(b[28]), C.uchar(b[29]), C.uchar(b[30]), C.uchar(b[31]),
	}
}

func bytesToChar16(b []byte) [16]C.uchar {
	for len(b) < 16 {
		b = append(b, byte(0))
	}
	return [16]C.uchar{
		C.uchar(b[0]), C.uchar(b[1]), C.uchar(b[2]), C.uchar(b[3]),
		C.uchar(b[4]), C.uchar(b[5]), C.uchar(b[6]), C.uchar(b[7]),
		C.uchar(b[8]), C.uchar(b[9]), C.uchar(b[10]), C.uchar(b[11]),
		C.uchar(b[12]), C.uchar(b[13]), C.uchar(b[14]), C.uchar(b[15]),
	}
}

func main() {}

var functions C.CK_FUNCTION_LIST = C.CK_FUNCTION_LIST{
	version: C.struct__CK_VERSION{
		major: 0,
		minor: 1,
	},
	C_Initialize:          C.CK_C_Initialize(C.C_Initialize),
	C_GetInfo:             C.CK_C_GetInfo(C.C_GetInfo),
	C_GetSlotList:         C.CK_C_GetSlotList(C.C_GetSlotList),
	C_GetSlotInfo:         C.CK_C_GetSlotInfo(C.C_GetSlotInfo),
	C_GetTokenInfo:        C.CK_C_GetTokenInfo(C.C_GetTokenInfo),
	C_OpenSession:         C.CK_C_OpenSession(C.C_OpenSession),
	C_CloseSession:        C.CK_C_CloseSession(C.C_CloseSession),
	C_FindObjectsInit:     C.CK_C_FindObjectsInit(C.C_FindObjectsInit),
	C_FindObjects:         C.CK_C_FindObjects(C.C_FindObjects),
	C_FindObjectsFinal:    C.CK_C_FindObjectsFinal(C.C_FindObjectsFinal),
	C_GetAttributeValue:   C.CK_C_GetAttributeValue(C.C_GetAttributeValue),
	C_SignInit:            C.CK_C_SignInit(C.C_SignInit),
	C_Sign:                C.CK_C_Sign(C.C_Sign),
	C_Finalize:            C.CK_C_Finalize(C.C_Finalize),
	C_GetMechanismList:    C.CK_C_GetMechanismList(C.C_GetMechanismList),
	C_GetMechanismInfo:    C.CK_C_GetMechanismInfo(C.C_GetMechanismInfo),
	C_InitToken:           C.CK_C_InitToken(C.C_InitToken),
	C_InitPIN:             C.CK_C_InitPIN(C.C_InitPIN),
	C_SetPIN:              C.CK_C_SetPIN(C.C_SetPIN),
	C_CloseAllSessions:    C.CK_C_CloseAllSessions(C.C_CloseAllSessions),
	C_GetSessionInfo:      C.CK_C_GetSessionInfo(C.C_GetSessionInfo),
	C_GetOperationState:   C.CK_C_GetOperationState(C.C_GetOperationState),
	C_SetOperationState:   C.CK_C_SetOperationState(C.C_SetOperationState),
	C_Login:               C.CK_C_Login(C.C_Login),
	C_Logout:              C.CK_C_Logout(C.C_Logout),
	C_CreateObject:        C.CK_C_CreateObject(C.C_CreateObject),
	C_CopyObject:          C.CK_C_CopyObject(C.C_CopyObject),
	C_DestroyObject:       C.CK_C_DestroyObject(C.C_DestroyObject),
	C_GetObjectSize:       C.CK_C_GetObjectSize(C.C_GetObjectSize),
	C_SetAttributeValue:   C.CK_C_SetAttributeValue(C.C_SetAttributeValue),
	C_EncryptInit:         C.CK_C_EncryptInit(C.C_EncryptInit),
	C_Encrypt:             C.CK_C_Encrypt(C.C_Encrypt),
	C_EncryptUpdate:       C.CK_C_EncryptUpdate(C.C_EncryptUpdate),
	C_EncryptFinal:        C.CK_C_EncryptFinal(C.C_EncryptFinal),
	C_DecryptInit:         C.CK_C_DecryptInit(C.C_DecryptInit),
	C_Decrypt:             C.CK_C_Decrypt(C.C_Decrypt),
	C_DecryptUpdate:       C.CK_C_DecryptUpdate(C.C_DecryptUpdate),
	C_DecryptFinal:        C.CK_C_DecryptFinal(C.C_DecryptFinal),
	C_DigestInit:          C.CK_C_DigestInit(C.C_DigestInit),
	C_Digest:              C.CK_C_Digest(C.C_Digest),
	C_DigestUpdate:        C.CK_C_DigestUpdate(C.C_DigestUpdate),
	C_DigestKey:           C.CK_C_DigestKey(C.C_DigestKey),
	C_DigestFinal:         C.CK_C_DigestFinal(C.C_DigestFinal),
	C_SignUpdate:          C.CK_C_SignUpdate(C.C_SignUpdate),
	C_SignFinal:           C.CK_C_SignFinal(C.C_SignFinal),
	C_SignRecoverInit:     C.CK_C_SignRecoverInit(C.C_SignRecoverInit),
	C_SignRecover:         C.CK_C_SignRecover(C.C_SignRecover),
	C_VerifyInit:          C.CK_C_VerifyInit(C.C_VerifyInit),
	C_Verify:              C.CK_C_Verify(C.C_Verify),
	C_VerifyUpdate:        C.CK_C_VerifyUpdate(C.C_VerifyUpdate),
	C_VerifyFinal:         C.CK_C_VerifyFinal(C.C_VerifyFinal),
	C_VerifyRecoverInit:   C.CK_C_VerifyRecoverInit(C.C_VerifyRecoverInit),
	C_VerifyRecover:       C.CK_C_VerifyRecover(C.C_VerifyRecover),
	C_DigestEncryptUpdate: C.CK_C_DigestEncryptUpdate(C.C_DigestEncryptUpdate),
	C_DecryptDigestUpdate: C.CK_C_DecryptDigestUpdate(C.C_DecryptDigestUpdate),
	C_SignEncryptUpdate:   C.CK_C_SignEncryptUpdate(C.C_SignEncryptUpdate),
	C_DecryptVerifyUpdate: C.CK_C_DecryptVerifyUpdate(C.C_DecryptVerifyUpdate),
	C_GenerateKey:         C.CK_C_GenerateKey(C.C_GenerateKey),
	C_GenerateKeyPair:     C.CK_C_GenerateKeyPair(C.C_GenerateKeyPair),
	C_WrapKey:             C.CK_C_WrapKey(C.C_WrapKey),
	C_UnwrapKey:           C.CK_C_UnwrapKey(C.C_UnwrapKey),
	C_DeriveKey:           C.CK_C_DeriveKey(C.C_DeriveKey),
	C_SeedRandom:          C.CK_C_SeedRandom(C.C_SeedRandom),
	C_GenerateRandom:      C.CK_C_GenerateRandom(C.C_GenerateRandom),
	C_GetFunctionStatus:   C.CK_C_GetFunctionStatus(C.C_GetFunctionStatus),
	C_CancelFunction:      C.CK_C_CancelFunction(C.C_CancelFunction),
	C_WaitForSlotEvent:    C.CK_C_WaitForSlotEvent(C.C_WaitForSlotEvent),
}
