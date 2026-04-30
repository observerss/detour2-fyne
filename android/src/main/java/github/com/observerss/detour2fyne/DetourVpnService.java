package github.com.observerss.detour2fyne;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Intent;
import android.net.VpnService;
import android.os.Build;
import android.os.ParcelFileDescriptor;
import android.util.Log;

public class DetourVpnService extends VpnService {
    private static final String TAG = "DetourVpnService";

    public static final String ACTION_START_VPN = "github.com.observerss.detour2fyne.action.START_VPN";
    public static final String ACTION_STOP_VPN = "github.com.observerss.detour2fyne.action.STOP_VPN";
    public static final String EXTRA_PROXY_SERVER = "proxyServer";

    private static final String CHANNEL_ID = "detour2_vpn_status";
    private static final int NOTIFICATION_ID = 2602;

    private ParcelFileDescriptor vpnInterface;
    private String proxyServer;

    static {
        System.loadLibrary("Detour2");
    }

    private static native void startTunForwarder(int fd);

    private static native void stopTunForwarder();

    @Override
    public void onCreate() {
        super.onCreate();
        createNotificationChannel();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if (intent == null) {
            return START_NOT_STICKY;
        }

        String action = intent.getAction();
        if (ACTION_STOP_VPN.equals(action)) {
            stopVpn();
            stopSelf();
            return START_NOT_STICKY;
        }

        proxyServer = intent.getStringExtra(EXTRA_PROXY_SERVER);
        Log.i(TAG, "Starting VPN service, proxy=" + proxyServer);
        startForeground(NOTIFICATION_ID, buildNotification());
        startVpn();
        return START_STICKY;
    }

    @Override
    public void onDestroy() {
        stopVpn();
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
        } catch (Throwable error) {
            Log.e(TAG, "Failed to start native TUN forwarder", error);
            stopVpn();
            stopSelf();
        }
    }

    private void stopVpn() {
        if (vpnInterface == null) {
            stopTunForwarder();
            stopForeground(true);
            return;
        }
        try {
            vpnInterface.close();
        } catch (Exception ignored) {
        } finally {
            vpnInterface = null;
            stopTunForwarder();
            stopForeground(true);
        }
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