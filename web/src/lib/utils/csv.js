/**
 * Utility functions for exporting data to CSV format
 */

/**
 * Convert server data to CSV format and trigger a download
 * @param {Array} servers - Array of server objects to export
 * @param {String} filename - Name of the CSV file to download
 */
export function exportServersToCSV(servers, filename = 'server-data.csv') {
  if (!servers || servers.length === 0) {
    console.error('No data to export');
    return;
  }

  // Define headers for CSV file
  const headers = [
    'URL',
    'Status',
    'Expected Status',
    'Creation Date',
    'Active',
    'Update Interval',
    'Security Risk Level',
    'Header Score',
    'Certificate Score',
    'Admin Risk',
    'API Risk'
  ];

  // Function to determine status from ping results
  const getStatus = (server) => {
    if (!server.ping_results || server.ping_results.length === 0) {
      return 'No data';
    }

    // Get the latest ping result
    const latestPing = server.ping_results[0];
    const isSuccess = latestPing.status_code === server.expected_status;

    return isSuccess ? 'Online' : 'Issue';
  };

  // Format date for CSV
  const formatDate = (dateString) => {
    if (!dateString) return 'N/A';
    const date = new Date(dateString);
    return date.toISOString();
  };

  // Create CSV rows
  const rows = servers.map(server => [
    server.url,
    getStatus(server),
    server.expected_status,
    formatDate(server.CreatedAt),
    server.active ? 'Yes' : 'No',
    server.update_interval,
    server.security_risk_level || 'N/A',
    server.header_score || 'N/A',
    server.cert_score || 'N/A',
    server.admin_risk || 'N/A',
    server.api_risk || 'N/A'
  ]);

  // Combine headers and rows
  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => {
      // Escape values that contain commas, quotes, or newlines
      const value = String(cell).replace(/"/g, '""');
      return /[,"\n\r]/.test(value) ? `"${value}"` : value;
    }).join(','))
  ].join('\n');

  // Create a blob and download link
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.setAttribute('href', url);
  link.setAttribute('download', filename);
  link.style.visibility = 'hidden';
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}
