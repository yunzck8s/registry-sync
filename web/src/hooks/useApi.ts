import { useState, useEffect, useCallback } from 'react';
import { message } from 'antd';

export function useApi<T>(
  apiFunc: () => Promise<{ data: T }>,
  deps: any[] = []
) {
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const response = await apiFunc();
      setData(response.data);
    } catch (err: any) {
      const errorMsg = err.response?.data?.error || err.message || '请求失败';
      setError(errorMsg);
      message.error(errorMsg);
    } finally {
      setLoading(false);
    }
  }, deps);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  return { data, loading, error, refetch: fetchData };
}

export function useAsyncAction<T = any>() {
  const [loading, setLoading] = useState(false);

  const execute = useCallback(
    async (
      apiFunc: () => Promise<T>,
      successMessage?: string
    ): Promise<T | null> => {
      try {
        setLoading(true);
        const result = await apiFunc();
        if (successMessage) {
          message.success(successMessage);
        }
        return result;
      } catch (err: any) {
        const errorMsg =
          err.response?.data?.error || err.message || '操作失败';
        message.error(errorMsg);
        return null;
      } finally {
        setLoading(false);
      }
    },
    []
  );

  return { loading, execute };
}
