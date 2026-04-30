//go:build android
// +build android

// Android系统代理配置
package proxy

/*
#include <jni.h>
#include <stdint.h>
#include <stdlib.h>

static int detour_check_exception(JNIEnv* env) {
	if ((*env)->ExceptionCheck(env)) {
		(*env)->ExceptionClear(env);
		return 1;
	}
	return 0;
}

static jstring detour_new_string(JNIEnv* env, const char* value) {
	if (value == NULL) {
		value = "";
	}
	return (*env)->NewStringUTF(env, value);
}

static int detour_service_available(JNIEnv* env, jobject context) {
	jclass context_class = (*env)->FindClass(env, "android/content/Context");
	if (context_class == NULL || detour_check_exception(env)) {
		return 0;
	}
	jmethodID get_class_loader = (*env)->GetMethodID(env, context_class, "getClassLoader", "()Ljava/lang/ClassLoader;");
	if (get_class_loader == NULL || detour_check_exception(env)) {
		return 0;
	}
	jobject class_loader = (*env)->CallObjectMethod(env, context, get_class_loader);
	if (class_loader == NULL || detour_check_exception(env)) {
		return 0;
	}
	jclass class_loader_class = (*env)->FindClass(env, "java/lang/ClassLoader");
	if (class_loader_class == NULL || detour_check_exception(env)) {
		return 0;
	}
	jmethodID load_class = (*env)->GetMethodID(env, class_loader_class, "loadClass", "(Ljava/lang/String;)Ljava/lang/Class;");
	if (load_class == NULL || detour_check_exception(env)) {
		return 0;
	}
	jstring service_name = detour_new_string(env, "github.com.observerss.detour2fyne.DetourVpnService");
	jobject service_class = (*env)->CallObjectMethod(env, class_loader, load_class, service_name);
	if (service_class == NULL || detour_check_exception(env)) {
		return 0;
	}
	return 1;
}

static int detour_request_notification_permission(uintptr_t jni_env, uintptr_t ctx) {
	JNIEnv* env = (JNIEnv*)jni_env;
	jobject context = (jobject)ctx;
	if (env == NULL || context == NULL) {
		return -1;
	}

	jclass build_version_class = (*env)->FindClass(env, "android/os/Build$VERSION");
	jfieldID sdk_int_field = (*env)->GetStaticFieldID(env, build_version_class, "SDK_INT", "I");
	jint sdk_int = (*env)->GetStaticIntField(env, build_version_class, sdk_int_field);
	if (detour_check_exception(env) || sdk_int < 33) {
		return 0;
	}

	jstring permission = detour_new_string(env, "android.permission.POST_NOTIFICATIONS");
	jclass context_class = (*env)->FindClass(env, "android/content/Context");
	jmethodID check_self_permission = (*env)->GetMethodID(env, context_class, "checkSelfPermission", "(Ljava/lang/String;)I");
	if (check_self_permission == NULL || detour_check_exception(env)) {
		return 0;
	}
	jint granted = (*env)->CallIntMethod(env, context, check_self_permission, permission);
	if (detour_check_exception(env) || granted == 0) {
		return 0;
	}

	jclass activity_class = (*env)->FindClass(env, "android/app/Activity");
	if (activity_class == NULL || detour_check_exception(env)) {
		return -2;
	}
	jmethodID request_permissions = (*env)->GetMethodID(env, activity_class, "requestPermissions", "([Ljava/lang/String;I)V");
	if (request_permissions == NULL || detour_check_exception(env)) {
		return -3;
	}
	jclass string_class = (*env)->FindClass(env, "java/lang/String");
	jobjectArray permissions = (*env)->NewObjectArray(env, 1, string_class, permission);
	(*env)->CallVoidMethod(env, context, request_permissions, permissions, 2604);
	if (detour_check_exception(env)) {
		return -4;
	}
	return 1;
}

static int detour_start_vpn(uintptr_t jni_env, uintptr_t ctx, const char* proxy_server) {
	JNIEnv* env = (JNIEnv*)jni_env;
	jobject context = (jobject)ctx;
	if (env == NULL || context == NULL) {
		return -1;
	}

	int notification_permission = detour_request_notification_permission(jni_env, ctx);
	if (notification_permission < 0) {
		return -12;
	}

	jclass vpn_class = (*env)->FindClass(env, "android/net/VpnService");
	if (vpn_class == NULL || detour_check_exception(env)) {
		return -2;
	}
	jmethodID prepare = (*env)->GetStaticMethodID(env, vpn_class, "prepare", "(Landroid/content/Context;)Landroid/content/Intent;");
	if (prepare == NULL || detour_check_exception(env)) {
		return -3;
	}
	jobject consent = (*env)->CallStaticObjectMethod(env, vpn_class, prepare, context);
	if (detour_check_exception(env)) {
		return -4;
	}

	jclass intent_class = (*env)->FindClass(env, "android/content/Intent");
	if (intent_class == NULL || detour_check_exception(env)) {
		return -5;
	}
	jclass context_class = (*env)->FindClass(env, "android/content/Context");
	if (context_class == NULL || detour_check_exception(env)) {
		return -6;
	}

	if (consent != NULL) {
		jclass activity_class = (*env)->FindClass(env, "android/app/Activity");
		if (activity_class == NULL || detour_check_exception(env)) {
			return -7;
		}
		jmethodID start_activity_for_result = (*env)->GetMethodID(env, activity_class, "startActivityForResult", "(Landroid/content/Intent;I)V");
		if (start_activity_for_result == NULL || detour_check_exception(env)) {
			return -7;
		}
		(*env)->CallVoidMethod(env, context, start_activity_for_result, consent, 2602);
		if (detour_check_exception(env)) {
			return -8;
		}
		return 1;
	}

	if (!detour_service_available(env, context)) {
		return -11;
	}

	jmethodID intent_ctor = (*env)->GetMethodID(env, intent_class, "<init>", "()V");
	jmethodID set_class_name = (*env)->GetMethodID(env, intent_class, "setClassName", "(Ljava/lang/String;Ljava/lang/String;)Landroid/content/Intent;");
	jmethodID set_action = (*env)->GetMethodID(env, intent_class, "setAction", "(Ljava/lang/String;)Landroid/content/Intent;");
	jmethodID put_extra = (*env)->GetMethodID(env, intent_class, "putExtra", "(Ljava/lang/String;Ljava/lang/String;)Landroid/content/Intent;");
	jmethodID start_service = (*env)->GetMethodID(env, context_class, "startService", "(Landroid/content/Intent;)Landroid/content/ComponentName;");
	if (intent_ctor == NULL || set_class_name == NULL || set_action == NULL || put_extra == NULL || start_service == NULL || detour_check_exception(env)) {
		return -9;
	}

	jobject intent = (*env)->NewObject(env, intent_class, intent_ctor);
	jstring package_name = detour_new_string(env, "github.com.observerss.detour2fyne");
	jstring service_name = detour_new_string(env, "github.com.observerss.detour2fyne.DetourVpnService");
	jstring action = detour_new_string(env, "github.com.observerss.detour2fyne.action.START_VPN");
	jstring extra_name = detour_new_string(env, "proxyServer");
	jstring proxy = detour_new_string(env, proxy_server);

	(*env)->CallObjectMethod(env, intent, set_class_name, package_name, service_name);
	(*env)->CallObjectMethod(env, intent, set_action, action);
	(*env)->CallObjectMethod(env, intent, put_extra, extra_name, proxy);
	(*env)->CallObjectMethod(env, context, start_service, intent);
	if (detour_check_exception(env)) {
		return -10;
	}
	return 0;
}

static int detour_stop_vpn(uintptr_t jni_env, uintptr_t ctx) {
	JNIEnv* env = (JNIEnv*)jni_env;
	jobject context = (jobject)ctx;
	if (env == NULL || context == NULL) {
		return -1;
	}

	jclass intent_class = (*env)->FindClass(env, "android/content/Intent");
	jclass context_class = (*env)->FindClass(env, "android/content/Context");
	if (intent_class == NULL || context_class == NULL || detour_check_exception(env)) {
		return -2;
	}
	jmethodID intent_ctor = (*env)->GetMethodID(env, intent_class, "<init>", "()V");
	jmethodID set_class_name = (*env)->GetMethodID(env, intent_class, "setClassName", "(Ljava/lang/String;Ljava/lang/String;)Landroid/content/Intent;");
	jmethodID stop_service = (*env)->GetMethodID(env, context_class, "stopService", "(Landroid/content/Intent;)Z");
	if (intent_ctor == NULL || set_class_name == NULL || stop_service == NULL || detour_check_exception(env)) {
		return -3;
	}

	jobject intent = (*env)->NewObject(env, intent_class, intent_ctor);
	jstring package_name = detour_new_string(env, "github.com.observerss.detour2fyne");
	jstring service_name = detour_new_string(env, "github.com.observerss.detour2fyne.DetourVpnService");
	(*env)->CallObjectMethod(env, intent, set_class_name, package_name, service_name);
	(*env)->CallBooleanMethod(env, context, stop_service, intent);
	if (detour_check_exception(env)) {
		return -4;
	}
	return 0;
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

var (
	statusMu sync.Mutex
	status   = "Android VPN 未启用"
)

// SetGlobalProxy 设置全局代理
func SetGlobalProxy(proxyServer string, bypasses ...string) error {
	proxyServer = strings.TrimSpace(proxyServer)
	if proxyServer == "" {
		return errors.New("代理服务器(proxyServer)配置为空")
	}
	setTunForwarderProxy(proxyServer)

	result, err := startVPN(proxyServer)
	if err != nil {
		setStatus("Android VPN 启动失败: " + err.Error())
		return err
	}
	if result == 1 {
		setStatus("需要授权 Android VPN；授权后请再次启用全局代理")
		return nil
	}
	setStatus("Android VPN Service 启动请求已发送")
	return nil
}

// Off 关闭代理
func Off() error {
	StopTunForwarder()
	if err := stopVPN(); err != nil {
		setStatus("Android VPN 停止失败: " + err.Error())
		return err
	}
	setStatus("Android VPN 已停止")
	return nil
}

func Status() string {
	statusMu.Lock()
	defer statusMu.Unlock()
	return status
}

func setStatus(value string) {
	statusMu.Lock()
	status = value
	statusMu.Unlock()
}

func startVPN(proxyServer string) (int, error) {
	var result int
	proxy := C.CString(proxyServer)
	defer C.free(unsafe.Pointer(proxy))

	err := driver.RunNative(func(native any) error {
		androidContext, ok := native.(*driver.AndroidContext)
		if !ok {
			return errors.New("无法获取 Android 运行上下文")
		}
		code := int(C.detour_start_vpn(C.uintptr_t(androidContext.Env), C.uintptr_t(androidContext.Ctx), proxy))
		if code < 0 {
			return vpnError(code)
		}
		result = code
		return nil
	})
	return result, err
}

func stopVPN() error {
	return driver.RunNative(func(native any) error {
		androidContext, ok := native.(*driver.AndroidContext)
		if !ok {
			return errors.New("无法获取 Android 运行上下文")
		}
		code := int(C.detour_stop_vpn(C.uintptr_t(androidContext.Env), C.uintptr_t(androidContext.Ctx)))
		if code < 0 {
			return vpnError(code)
		}
		return nil
	})
}

func vpnError(code int) error {
	if code == -11 {
		return errors.New("APK 尚未合并 DetourVpnService；需要先接入 Android Java/DEX 打包步骤")
	}
	return fmt.Errorf("Android VPN JNI 调用失败(%d)，请确认 APK 已合并 DetourVpnService", code)
}
