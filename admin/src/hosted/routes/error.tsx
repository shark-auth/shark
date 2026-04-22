import React from 'react'

interface ErrorPageProps {
  code: number
  message: string
}

export function ErrorPage({ code, message }: ErrorPageProps) {
  return <div>error stub — {code}: {message}</div>
}
