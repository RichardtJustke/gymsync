"use client";

import React, { useEffect, useRef, useState } from "react";

// BarcodeDetector ainda não está nos tipos do TypeScript
type Detector = { detect: (src: CanvasImageSource) => Promise<{ rawValue: string }[]> };

/** Lê o QR da recepção pela câmera (onde o navegador suporta) e devolve o texto lido */
export default function QrScanner({ onScan, onCancel }: { onScan: (value: string) => void; onCancel: () => void }) {
  const video = useRef<HTMLVideoElement>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let stream: MediaStream | null = null;
    let raf = 0;
    let stopped = false;

    (async () => {
      const Ctor = (window as unknown as { BarcodeDetector?: new (o: { formats: string[] }) => Detector }).BarcodeDetector;
      if (!Ctor) {
        setError("Seu navegador não lê QR pela câmera. Digite o código de 6 números que aparece na tela da recepção.");
        return;
      }
      try {
        const detector = new Ctor({ formats: ["qr_code"] });
        stream = await navigator.mediaDevices.getUserMedia({ video: { facingMode: "environment" } });
        if (stopped || !video.current) return;
        video.current.srcObject = stream;
        await video.current.play();
        const tick = async () => {
          if (stopped || !video.current) return;
          try {
            const found = await detector.detect(video.current);
            if (found[0]?.rawValue) {
              stopped = true;
              onScan(found[0].rawValue);
              return;
            }
          } catch {
            /* quadro sem imagem ainda */
          }
          raf = requestAnimationFrame(tick);
        };
        tick();
      } catch {
        setError("Não consegui abrir a câmera. Libere o acesso nas permissões do navegador ou digite o código.");
      }
    })();

    return () => {
      stopped = true;
      cancelAnimationFrame(raf);
      stream?.getTracks().forEach((t) => t.stop());
    };
  }, [onScan]);

  return (
    <div className="space-y-3">
      {error ? (
        <p className="text-sm text-ink-200 p-4 rounded-2xl bg-ink-750">{error}</p>
      ) : (
        <div className="relative rounded-3xl overflow-hidden bg-black aspect-square">
          <video ref={video} playsInline muted className="w-full h-full object-cover" />
          <div className="absolute inset-8 rounded-3xl border-2 border-lime/80 pointer-events-none" />
        </div>
      )}
      <button className="btn-dark w-full" onClick={onCancel}>Fechar câmera</button>
    </div>
  );
}
