"use client";

import { useEffect, useRef } from "react";
import { usePathname, useRouter } from "next/navigation";
import { Capacitor } from "@capacitor/core";
import { PushNotifications } from "@capacitor/push-notifications";
import { api, getToken } from "@/services/api";

// No app nativo (Capacitor) pede permissão de notificação e envia o token FCM/APNs ao backend.
// Re-tenta a cada troca de rota enquanto houver sessão (cobre o login logo após abrir o app).
export default function PushRegistrar() {
  const pathname = usePathname();
  const router = useRouter();
  const sent = useRef<string | null>(null);
  const listening = useRef(false);

  useEffect(() => {
    if (!Capacitor.isNativePlatform() || !getToken()) return;
    const platform = Capacitor.getPlatform();

    (async () => {
      try {
        if (!listening.current) {
          listening.current = true;
          await PushNotifications.addListener("registration", (t) => {
            if (sent.current === t.value) return;
            sent.current = t.value;
            api.registerDevice(t.value, platform).catch(() => { sent.current = null; });
          });
          await PushNotifications.addListener("pushNotificationActionPerformed", (a) => {
            const link = a.notification.data?.link;
            if (typeof link === "string" && link.startsWith("/")) router.push(link);
          });
        }
        let perm = await PushNotifications.checkPermissions();
        if (perm.receive === "prompt") perm = await PushNotifications.requestPermissions();
        if (perm.receive === "granted") await PushNotifications.register();
      } catch {
        /* push é opcional: falha não deve afetar o app */
      }
    })();
  }, [pathname, router]);

  return null;
}
