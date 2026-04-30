//go:build android
// +build android

package main

/*
#include <jni.h>
*/
import "C"

import "github.com/observerss/detour2-fyne/run/proxy"

//export Java_github_com_observerss_detour2fyne_DetourVpnService_startTunForwarder
func Java_github_com_observerss_detour2fyne_DetourVpnService_startTunForwarder(env *C.JNIEnv, clazz C.jclass, fd C.jint) {
	_ = proxy.StartTunForwarder(int(fd))
}

//export Java_github_com_observerss_detour2fyne_DetourVpnService_stopTunForwarder
func Java_github_com_observerss_detour2fyne_DetourVpnService_stopTunForwarder(env *C.JNIEnv, clazz C.jclass) {
	proxy.StopTunForwarder()
}
