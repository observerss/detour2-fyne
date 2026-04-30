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

//export Java_github_com_observerss_detour2fyne_DetourVpnService_notifyVpnStarted
func Java_github_com_observerss_detour2fyne_DetourVpnService_notifyVpnStarted(env *C.JNIEnv, clazz C.jclass) {
	proxy.NotifyVPNStarted()
}

//export Java_github_com_observerss_detour2fyne_DetourVpnService_notifyVpnStopped
func Java_github_com_observerss_detour2fyne_DetourVpnService_notifyVpnStopped(env *C.JNIEnv, clazz C.jclass) {
	proxy.NotifyVPNStopped()
}

//export Java_github_com_observerss_detour2fyne_DetourVpnService_notifyVpnRevoked
func Java_github_com_observerss_detour2fyne_DetourVpnService_notifyVpnRevoked(env *C.JNIEnv, clazz C.jclass) {
	proxy.NotifyVPNRevoked()
}
