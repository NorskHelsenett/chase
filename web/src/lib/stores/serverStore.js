import { writable, derived } from 'svelte/store';
import { browser } from '$app/environment';

// Initialize store with cached data from localStorage if available
const initialData = browser && localStorage.getItem('cachedServers') 
  ? JSON.parse(localStorage.getItem('cachedServers')) 
  : [];

// Main server data store
const serverStore = writable({
  servers: initialData,
  isLoading: false,
  lastUpdated: browser && localStorage.getItem('serversLastUpdated')
    ? new Date(localStorage.getItem('serversLastUpdated'))
    : null,
  error: null
});

// Derived store for statistics
export const serverStats = derived(serverStore, ($store) => {
  return $store.servers.reduce((acc, server) => {
    const sortedPings = [...(server.ping_results || [])].sort((a, b) =>
      new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
    );
    const latestPing = sortedPings[0];

    if (latestPing) {
      if (latestPing.status_code === server.expected_status) {
        acc.up += 1;
      } else {
        acc.down += 1;
      }

      // Safely check for TLS validity if the field exists
      if (latestPing.tls_valid === false) {
        acc.criticalRisks += 1;
      }

      // Only check cert_expiry_date if it exists
      if (latestPing.cert_expiry_date) {
        try {
          const certExpiryDate = new Date(latestPing.cert_expiry_date);
          const daysUntilExpiry = Math.floor(
            (certExpiryDate.getTime() - new Date().getTime()) / (1000 * 60 * 60 * 24)
          );

          if (daysUntilExpiry < 30 && daysUntilExpiry > 0) {
            acc.highRisks += 1;
          }
        } catch (error) {
          // Ignore invalid dates
          console.warn("Invalid cert expiry date:", latestPing.cert_expiry_date, error);
        }
      }
    } else {
      acc.down += 1;
    }

    return acc;
  }, {
    up: 0,
    down: 0,
    criticalRisks: 0,
    highRisks: 0
  });
});

// Save to localStorage whenever the store changes
serverStore.subscribe(state => {
  if (browser && state.servers.length > 0) {
    try {
      localStorage.setItem('cachedServers', JSON.stringify(state.servers));
      localStorage.setItem('serversLastUpdated', state.lastUpdated ? state.lastUpdated.toISOString() : new Date().toISOString());
    } catch (error) {
      console.warn('Failed to save servers to localStorage:', error);
    }
  }
});

// Helper functions to interact with the store
export const serverStoreActions = {
  // Load servers from API with optional filter
  async loadServers(filter = null, force = false) {
    // Get current state
    let currentState;
    serverStore.update(state => {
      currentState = state;
      return { ...state, isLoading: true, error: null };
    });
    
    // If we have cached data and it's recent (less than 5 minutes old) and not forced refresh
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
    if (!force && currentState.lastUpdated && new Date(currentState.lastUpdated) > fiveMinutesAgo) {
      // Just mark as not loading and return cached data
      serverStore.update(state => ({
        ...state,
        isLoading: false
      }));
      return currentState.servers;
    }
    
    try {
      // Build URL with query parameters
      const url = new URL('/api/servers', window.location.origin);
      if (filter !== null) {
        url.searchParams.set('active', filter);
      }

      const response = await fetch(url);
      if (!response.ok) throw new Error('Failed to fetch servers');
      
      const servers = await response.json();
      
      // Preserve existing security data if available and merge with new server data
      const mergedServers = servers.map(server => {
        const existingServer = currentState.servers.find(s => s.ID === server.ID);
        if (existingServer) {
          return {
            ...server,
            // ping_results are now from the API response
            security: existingServer.security || null
          };
        }
        return {
          ...server,
          security: null
        };
      });
      
      // Sort servers
      mergedServers.sort((a, b) => {
        const nameA = a.name || a.url || '';
        const nameB = b.name || b.url || '';
        return nameA.localeCompare(nameB);
      });
      
      serverStore.update(state => ({
        ...state,
        servers: mergedServers,
        isLoading: false,
        lastUpdated: new Date()
      }));
      
      return mergedServers;
    } catch (error) {
      serverStore.update(state => ({
        ...state,
        isLoading: false,
        error: error.message
      }));
      console.error('Failed to fetch server data:', error);
      return currentState.servers;
    }
  },

  // Load more ping results for a specific server if needed (beyond what's in the initial data)
  async loadMoreServerPings(serverId) {
    try {
      let currentState;
      serverStore.subscribe(state => {
        currentState = state;
      })();
      
      const existingServer = currentState.servers.find(s => s.ID === serverId);
      
      // If we already have some ping results, check if we need more
      if (existingServer && existingServer.ping_results && existingServer.ping_results.length > 0) {
        // If we have 10 or more recent pings, don't fetch more
        if (existingServer.ping_results.length >= 10) {
          return existingServer.ping_results;
        }
      }
      
      const response = await fetch(`/api/servers/${serverId}/pings`);
      if (!response.ok) throw new Error(`Failed to fetch ping results for server ${serverId}`);
      
      const pingResults = await response.json();
      
      // Update the specific server with full ping results
      serverStore.update(state => {
        const updatedServers = state.servers.map(server => {
          if (server.ID === serverId) {
            return { ...server, ping_results: pingResults };
          }
          return server;
        });
        
        return {
          ...state,
          servers: updatedServers
        };
      });
      
      return pingResults;
    } catch (error) {
      console.error(`Failed to fetch more ping results for server ${serverId}:`, error);
      return [];
    }
  },
  
  // Load security report for a specific server
  async loadServerSecurityReport(serverId) {
    try {
      // Check if we already have security data that's recent
      let currentState;
      serverStore.subscribe(state => {
        currentState = state;
      })();
      
      const existingServer = currentState.servers.find(s => s.ID === serverId);
      if (existingServer && existingServer.security && existingServer.security.scanTimestamp) {
        // Check if security data is recent (less than 24 hours old)
        const twentyFourHoursAgo = new Date(Date.now() - 24 * 60 * 60 * 1000);
        if (new Date(existingServer.security.scanTimestamp) > twentyFourHoursAgo) {
          // Use cached data
          return existingServer.security;
        }
      }
      
      const response = await fetch(`/api/servers/${serverId}/report`);
      if (!response.ok) throw new Error(`Failed to fetch security report for server ${serverId}`);
      
      const securityReport = await response.json();
      
      // Extract only the data needed for the view
      const securityData = {
        headerRisk: securityReport.headers?.score || '',
        certRisk: securityReport.certificate?.grade || '',
        adminRisk: securityReport.adminPages?.risk || '',
        apiRisk: securityReport.swagger?.risk || '',
        scanTimestamp: securityReport.scanTimestamp || ''
      };
      
      // Update the specific server with security data
      serverStore.update(state => {
        const updatedServers = state.servers.map(server => {
          if (server.ID === serverId) {
            return { ...server, security: securityData };
          }
          return server;
        });
        
        return {
          ...state,
          servers: updatedServers
        };
      });
      
      return securityData;
    } catch (error) {
      console.error(`Failed to fetch security report for server ${serverId}:`, error);
      return null;
    }
  },

  // Add a new server
  async addServer(serverData) {
    try {
      const response = await fetch('/api/servers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(serverData)
      });
      
      if (!response.ok) throw new Error('Failed to add server');
      
      const newServer = await response.json();
      
      // Add the new server to the store
      serverStore.update(state => ({
        ...state,
        servers: [...state.servers, newServer],
        lastUpdated: new Date()
      }));
      
      return newServer;
    } catch (error) {
      console.error('Failed to add server:', error);
      throw error;
    }
  },

  // Update a server
  async updateServer(serverId, serverData) {
    try {
      const response = await fetch(`/api/servers/${serverId}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(serverData)
      });
      
      if (!response.ok) throw new Error(`Failed to update server ${serverId}`);
      
      const updatedServer = await response.json();
      
      // Update the server in the store
      serverStore.update(state => {
        const updatedServers = state.servers.map(server => {
          if (server.ID === serverId) {
            return updatedServer;
          }
          return server;
        });
        
        return {
          ...state,
          servers: updatedServers,
          lastUpdated: new Date()
        };
      });
      
      return updatedServer;
    } catch (error) {
      console.error(`Failed to update server ${serverId}:`, error);
      throw error;
    }
  },

  // Delete a server
  async deleteServer(serverId) {
    try {
      const response = await fetch(`/api/servers/${serverId}`, {
        method: 'DELETE'
      });
      
      if (!response.ok) throw new Error(`Failed to delete server ${serverId}`);
      
      // Remove the server from the store
      serverStore.update(state => ({
        ...state,
        servers: state.servers.filter(server => server.ID !== serverId),
        lastUpdated: new Date()
      }));
      
      return true;
    } catch (error) {
      console.error(`Failed to delete server ${serverId}:`, error);
      throw error;
    }
  },

  // Force check a server
  async forceCheckServer(serverId) {
    try {
      const response = await fetch(`/api/servers/${serverId}/force-check`, {
        method: 'POST'
      });
      
      if (!response.ok) throw new Error(`Failed to force check server ${serverId}`);
      
      // Refresh the server data
      await this.loadServers(null, true);
      
      return true;
    } catch (error) {
      console.error(`Failed to force check server ${serverId}:`, error);
      throw error;
    }
  },

  // Filter servers by search term
  filterServers(searchTerm) {
    if (!searchTerm) {
      return serverStore.subscribe(state => state.servers);
    }
    
    const term = searchTerm.toLowerCase();
    return derived(serverStore, $store => 
      $store.servers.filter(server =>
        server.url.toLowerCase().includes(term) ||
        (server.comment && server.comment.toLowerCase().includes(term))
      )
    );
  },

  // Get loading state
  isLoading() {
    return derived(serverStore, $store => $store.isLoading);
  },
  
  // Get error state  
  getError() {
    return derived(serverStore, $store => $store.error);
  }
};

// Export the server store for direct subscription
export const servers = derived(serverStore, $store => $store.servers);
export const isLoading = derived(serverStore, $store => $store.isLoading);
