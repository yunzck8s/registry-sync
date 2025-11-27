import { useEffect, useState } from 'react';
import { wsClient } from '../api/websocket';
import type { WSMessage } from '../types';

export function useWebSocket() {
  const [connected, setConnected] = useState(false);
  const [messages, setMessages] = useState<WSMessage[]>([]);

  useEffect(() => {
    // Connect to WebSocket
    const wsUrl = `ws://${window.location.hostname}:${
      window.location.port || '8080'
    }/api/v1/ws`;

    wsClient.connect(wsUrl);
    setConnected(true);

    // Subscribe to messages
    const unsubscribe = wsClient.subscribe((message) => {
      setMessages((prev) => [...prev, message]);
    });

    // Cleanup
    return () => {
      unsubscribe();
      wsClient.disconnect();
      setConnected(false);
    };
  }, []);

  const clearMessages = () => {
    setMessages([]);
  };

  return {
    connected,
    messages,
    clearMessages,
  };
}

export default useWebSocket;
