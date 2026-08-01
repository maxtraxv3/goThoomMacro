//go:build cgo && windows

package vosk

/*
#include <windows.h>
#include <stdlib.h>

static void *stt_model_new(void *lib, const char *path) {
	typedef void *(*fn_t)(const char *);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_model_new");
	return fn(path);
}
static void stt_model_free(void *lib, void *m) {
	typedef void (*fn_t)(void *);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_model_free");
	fn(m);
}
static void *stt_recognizer_new(void *lib, void *m, float rate) {
	typedef void *(*fn_t)(void *, float);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_recognizer_new");
	return fn(m, rate);
}
static void stt_recognizer_free(void *lib, void *r) {
	typedef void (*fn_t)(void *);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_recognizer_free");
	fn(r);
}
static int stt_accept_waveform_f(void *lib, void *r, const float *data, int len) {
	typedef int (*fn_t)(void *, const float *, int);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_recognizer_accept_waveform_f");
	return fn(r, data, len);
}
static const char *stt_result(void *lib, void *r) {
	typedef const char *(*fn_t)(void *);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_recognizer_result");
	return fn(r);
}
static const char *stt_partial_result(void *lib, void *r) {
	typedef const char *(*fn_t)(void *);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_recognizer_partial_result");
	return fn(r);
}
static const char *stt_final_result(void *lib, void *r) {
	typedef const char *(*fn_t)(void *);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_recognizer_final_result");
	return fn(r);
}
static void stt_recognizer_reset(void *lib, void *r) {
	typedef void (*fn_t)(void *);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_recognizer_reset");
	fn(r);
}
static void stt_set_log_level(void *lib, int level) {
	typedef void (*fn_t)(int);
	fn_t fn = (fn_t)GetProcAddress((HMODULE)lib, "vosk_set_log_level");
	fn(level);
}
*/
import "C"

import "unsafe"

var libHandle unsafe.Pointer

func loadLibrary(path string) unsafe.Pointer {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return C.LoadLibraryA(cpath)
}

func closeLibrary() {
	if libHandle != nil {
		C.FreeLibrary(libHandle)
	}
}

func modelNew(path string) unsafe.Pointer {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	return unsafe.Pointer(C.stt_model_new(libHandle, cpath))
}

func modelFree(m unsafe.Pointer) {
	C.stt_model_free(libHandle, m)
}

func recognizerNew(m unsafe.Pointer, rate float32) unsafe.Pointer {
	return unsafe.Pointer(C.stt_recognizer_new(libHandle, m, C.float(rate)))
}

func recognizerFree(r unsafe.Pointer) {
	C.stt_recognizer_free(libHandle, r)
}

func acceptWaveformF(r unsafe.Pointer, samples []float32) bool {
	return C.stt_accept_waveform_f(libHandle, r, (*C.float)(unsafe.Pointer(&samples[0])), C.int(len(samples))) != 0
}

func result(r unsafe.Pointer) string {
	return C.GoString(C.stt_result(libHandle, r))
}

func partialResult(r unsafe.Pointer) string {
	return C.GoString(C.stt_partial_result(libHandle, r))
}

func finalResult(r unsafe.Pointer) string {
	return C.GoString(C.stt_final_result(libHandle, r))
}

func recognizerReset(r unsafe.Pointer) {
	C.stt_recognizer_reset(libHandle, r)
}

func setLogLevel(level int) {
	C.stt_set_log_level(libHandle, C.int(level))
}
