// @ts-nocheck
import React from 'react'

export function useURLParam(key, defaultValue = '') {
  const getParam = () => {
    const params = new URLSearchParams(window.location.search);
    return params.get(key) ?? defaultValue;
  };

  const [value, setValue] = React.useState(getParam);

  React.useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    if (value === defaultValue || value === '' || value === null) {
      params.delete(key);
    } else {
      params.set(key, String(value));
    }
    const search = params.toString();
    const url = window.location.pathname + (search ? '?' + search : '');
    window.history.replaceState(null, '', url);
  }, [key, value, defaultValue]);

  return [value, setValue];
}

// useTabParam — convenience wrapper for tab state persisted in ?tab=
// Pages call this instead of React.useState('tab1') so refresh keeps the tab.
// Default is the first tab; setting back to default removes the param.
export function useTabParam(defaultTab) {
  return useURLParam('tab', defaultTab);
}
