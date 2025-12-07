export function stripSourcesFromContent(content: string): string {
  const lower = content.toLowerCase();
  const marker = "\nsources";
  const idx = lower.lastIndexOf(marker);

  if (idx === -1) {
    return content.trim();
  }

  return content.slice(0, idx).trimEnd();
}
