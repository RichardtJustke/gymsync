// Prefixo para servir o app em um subpath (ex.: GitHub Pages em /gymsync).
// Vazio no deploy raiz (servidor/Capacitor).
export const BASE_PATH = process.env.NEXT_PUBLIC_BASE_PATH || "";

/** Caminho para um arquivo de /public respeitando o basePath. */
export function asset(path: string): string {
  return `${BASE_PATH}${path.startsWith("/") ? path : `/${path}`}`;
}
