'use client'

import { useRouter } from 'next/navigation'
import { useEffect } from 'react'

export default function SentPage() {
  const router = useRouter()
  useEffect(() => {
    router.replace('/messages?tab=sent')
  }, [router])
  return null
}
