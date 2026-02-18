// isImageFile — checks whether a File object has an image MIME type
export function isImageFile(file: File): boolean {
  return file.type.startsWith('image/');
}
