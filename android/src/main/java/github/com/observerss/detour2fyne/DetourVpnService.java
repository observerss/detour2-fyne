package github.com.observerss.detour2fyne;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Intent;
import android.content.SharedPreferences;
import android.net.VpnService;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.os.ParcelFileDescriptor;
import android.util.Log;

public class DetourVpnService extends VpnService {
    private static final String TAG = "DetourVpnService";

    public static final String ACTION_START_VPN = "github.com.observerss.detour2fyne.action.START_VPN";
    public static final String ACTION_STOP_VPN = "github.com.observerss.detour2fyne.action.STOP_VPN";
    public static final String EXTRA_PROXY_SERVER = "proxyServer";

    private static final String CHANNEL_ID = "detour2_vpn_status";
    private static final int NOTIFICATION_ID = 2602;

    private static final String PREFS_NAME = "detour2_vpn";
    private static final String PREF_DESIRED = "desired";
    private static final String PREF_PROXY_SERVER = "proxyServer";
    private static final long RESTART_DELAY_MS = 2000L;

    private ParcelFileDescriptor vpnInterface;
    private String proxyServer;
    private final Handler handler = new Handler(Looper.getMainLooper());

    static {
        System.loadLibrary("Detour2");
    }

    private static native void startTunForwarder(int fd);

    private static native void stopTunForwarder();

    private static native void notifyVpnStarted();

    private static native void notifyVpnStopped();

    private static native void notifyVpnRevoked();

    @Override
    public void onCreate() {
        super.onCreate();
        createNotificationChannel();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if (intent == null) {
            if (shouldRun()) {
                proxyServer = loadProxyServer();
                if (proxyServer != null && proxyServer.length() > 0) {
                    Log.i(TAG, "Restarting sticky VPN service, proxy=" + proxyServer);
                    startForeground(NOTIFICATION_ID, buildNotification());
                    startVpn();
                    return START_STICKY;
                }
            }
            notifyVpnStopped();
            return START_NOT_STICKY;
        }

        String action = intent.getAction();
        if (ACTION_STOP_VPN.equals(action)) {
            setShouldRun(false);
            stopVpn();
            stopSelf();
            return START_NOT_STICKY;
        }

        proxyServer = intent.getStringExtra(EXTRA_PROXY_SERVER);
        if (proxyServer == null || proxyServer.length() == 0) {
            proxyServer = loadProxyServer();
        }
        saveDesiredState(proxyServer);
        Log.i(TAG, "Starting VPN service, proxy=" + proxyServer);
        startForeground(NOTIFICATION_ID, buildNotification());
        startVpn();
        return START_STICKY;
    }

    @Override
    public void onRevoke() {
        Log.w(TAG, "VPN revoked by system");
        notifyVpnRevoked();
        stopVpn();
        if (shouldRun()) {
            scheduleRestart();
        }
        super.onRevoke();
    }

    @Override
    public void onDestroy() {
        boolean restart = shouldRun();
        stopVpn();
        if (restart) {
            scheduleRestart();
        }
        super.onDestroy();
    }

    private void startVpn() {
        if (vpnInterface != null) {
            return;
        }

        Builder builder = new Builder()
                .setSession("Detour2")
                .addAddress("10.21.0.2", 32)
                .addRoute("0.0.0.0", 0)
                .addDnsServer("1.1.1.1")
                .setMtu(1500);

        try {
            builder.addDisallowedApplication(getPackageName());
        } catch (Exception ignored) {
        }

        vpnInterface = builder.establish();
        if (vpnInterface == null) {
            Log.w(TAG, "VPN establish returned null; user consent may be missing");
            return;
        }

        try {
            int fd = vpnInterface.detachFd();
            vpnInterface = null;
            Log.i(TAG, "VPN TUN established, fd=" + fd);
            startTunForwarder(fd);
            notifyVpnStarted();
        } catch (Throwable error) {
            Log.e(TAG, "Failed to start native TUN forwarder", error);
            stopVpn();
            stopSelf();
        }
    }

    private void stopVpn() {
        if (vpnInterface == null) {
            stopTunForwarder();
            notifyVpnStopped();
            stopForeground(true);
            return;
        }
        try {
            vpnInterface.close();
        } catch (Exception ignored) {
        } finally {
            vpnInterface = null;
            stopTunForwarder();
            notifyVpnStopped();
            stopForeground(true);
        }
    }

    private void scheduleRestart() {
        final String savedProxy = loadProxyServer();
        if (savedProxy == null || savedProxy.length() == 0) {
            return;
        }
        handler.postDelayed(new Runnable() {
            @Override
            public void run() {
                if (!shouldRun()) {
                    return;
                }
                Intent restartIntent = new Intent(DetourVpnService.this, DetourVpnService.class);
                restartIntent.setAction(ACTION_START_VPN);
                restartIntent.putExtra(EXTRA_PROXY_SERVER, savedProxy);
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                    startForegroundService(restartIntent);
                } else {
                    startService(restartIntent);
                }
            }
        }, RESTART_DELAY_MS);
    }

    private void saveDesiredState(String proxy) {
        SharedPreferences.Editor editor = getSharedPreferences(PREFS_NAME, MODE_PRIVATE).edit()
                .putBoolean(PREF_DESIRED, true);
        if (proxy != null && proxy.length() > 0) {
            editor.putString(PREF_PROXY_SERVER, proxy);
        }
        editor.apply();
    }

    private void setShouldRun(boolean value) {
        SharedPreferences.Editor editor = getSharedPreferences(PREFS_NAME, MODE_PRIVATE).edit()
                .putBoolean(PREF_DESIRED, value);
        if (!value) {
            editor.remove(PREF_PROXY_SERVER);
        }
        editor.apply();
    }

    private boolean shouldRun() {
        return getSharedPreferences(PREFS_NAME, MODE_PRIVATE).getBoolean(PREF_DESIRED, false);
    }

    private String loadProxyServer() {
        return getSharedPreferences(PREFS_NAME, MODE_PRIVATE).getString(PREF_PROXY_SERVER, "");
    }

    private Notification buildNotification() {
        String text = proxyServer == null || proxyServer.length() == 0
                ? "VPN 已开启，正在保护你的网络"
                : "已通过 " + proxyServer + " 转发网络流量";

        Notification.Builder builder = Build.VERSION.SDK_INT >= Build.VERSION_CODES.O
                ? new Notification.Builder(this, CHANNEL_ID)
                : new Notification.Builder(this);

        Intent stopIntent = new Intent(this, DetourVpnService.class);
        stopIntent.setAction(ACTION_STOP_VPN);
        PendingIntent stopPendingIntent = PendingIntent.getService(
                this,
                2603,
                stopIntent,
                pendingIntentFlags()
        );

        Notification.Action stopAction = new Notification.Action.Builder(
                android.R.drawable.button_onoff_indicator_on,
                "关闭 VPN",
                stopPendingIntent
        ).build();

        builder
                .setContentTitle("Detour2 VPN · 已开启")
                .setContentText(text)
                .setSmallIcon(android.R.drawable.presence_online)
                .setOngoing(true)
                .setAutoCancel(false)
                .setLocalOnly(true)
                .setPriority(Notification.PRIORITY_DEFAULT)
                .setOnlyAlertOnce(true)
                .setShowWhen(false)
                .addAction(stopAction);

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP) {
            builder.setVisibility(Notification.VISIBILITY_PUBLIC);
            builder.setCategory(Notification.CATEGORY_STATUS);
        }

        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            builder.setStyle(new Notification.BigTextStyle().bigText(text));
        }

        return builder.build();
    }

    private int pendingIntentFlags() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M) {
            return PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE;
        }
        return PendingIntent.FLAG_UPDATE_CURRENT;
    }

    private void createNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return;
        }
        NotificationManager manager = getSystemService(NotificationManager.class);
        if (manager != null) {
            manager.deleteNotificationChannel("detour2_vpn");
            NotificationChannel channel = new NotificationChannel(
                    CHANNEL_ID,
                    "Detour2 VPN 常驻状态",
                    NotificationManager.IMPORTANCE_DEFAULT
            );
            channel.setDescription("Detour2 VPN 运行时常驻通知");
            channel.setShowBadge(false);
            manager.createNotificationChannel(channel);
        }
    }
}