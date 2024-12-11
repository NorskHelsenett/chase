export function copyToClipboard() {
  navigator.clipboard.writeText(apiToken);
  copied = true;
  setTimeout(() => {
    copied = false;
  }, 2000);
}

export function getRiskColor(risk) {
  switch (risk?.toLowerCase()) {
    case 'critical':
    case 'f':
      return 'text-red-600';
    case 'high':
    case 'c':
      return 'text-red-500';
    case 'medium':
    case 'b':
    case 'b+':
      return 'text-yellow-500';
    case 'low':
    case 'a':
    case 'a+':
      return 'text-green-500';
    default:
      return 'text-gray-500';
  }
}