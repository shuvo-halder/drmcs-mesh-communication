import { useEffect, useRef, useCallback } from 'react';

/**
 * Custom hook for consuming Server-Sent Events (SSE) from the analytics API.
 * Provides real-time updates without external dependencies.
 */
export default function useEventStream(url, onEvent) {
  const eventSourceRef = useRef(null);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    try {
      const es = new EventSource(url);
      eventSourceRef.current = es;

      es.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (onEventRef.current) {
            onEventRef.current(data);
          }
        } catch (err) {
          console.error('SSE parse error:', err);
        }
      };

      es.onerror = () => {
        console.warn('SSE connection error, reconnecting in 3s...');
        es.close();
        setTimeout(() => connect(), 3000);
      };

      es.onopen = () => {
        console.log('SSE connected:', url);
      };
    } catch (err) {
      console.error('SSE connection failed:', err);
      setTimeout(() => connect(), 5000);
    }
  }, [url]);

  useEffect(() => {
    connect();
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, [connect]);

  return null;
}