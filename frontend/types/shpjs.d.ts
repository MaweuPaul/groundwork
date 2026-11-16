declare module 'shpjs' {
  type BufferLike = ArrayBuffer | ArrayBufferView | DataView;

  export function parseShp(shp: BufferLike, prj?: string | BufferLike): unknown[];
  export function parseDbf(dbf: BufferLike, cpg?: string | BufferLike): Record<string, unknown>[];
}
