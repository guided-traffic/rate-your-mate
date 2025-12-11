export const environment = {
  production: true,
  apiUrl: '/api/v1',
  wsUrl: 'wss://' + (typeof window !== 'undefined' ? window.location.host : '') + '/api/v1/ws',
  version: '__VERSION__'
};
