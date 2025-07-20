import React, { useState, useRef, useEffect } from 'react'
import { Share2, Check, Copy } from 'lucide-react'
import { Button } from '../ui/button'
import { useShareableLink } from '../../hooks/useNavigation'
import { useToast } from '../../contexts/ToastContext'

export function ShareLinkButton() {
  const { shareableLink, copyToClipboard } = useShareableLink()
  const [copied, setCopied] = useState(false)
  const toast = useToast()
  const timeoutRef = useRef<NodeJS.Timeout | null>(null)

  // Clean up timeout on unmount
  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const handleShare = async () => {
    const success = await copyToClipboard()
    if (success) {
      setCopied(true)
      toast.success('Link Copied!', 'The shareable link has been copied to your clipboard')
      
      // Clear any existing timeout
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
      
      // Set new timeout with cleanup reference
      timeoutRef.current = setTimeout(() => {
        setCopied(false)
        timeoutRef.current = null
      }, 2000)
    } else {
      toast.error('Copy Failed', 'Failed to copy link to clipboard')
    }
  }

  return (
    <Button
      variant="outline"
      size="sm"
      onClick={handleShare}
      className="flex items-center gap-2"
      title="Share this view"
    >
      {copied ? (
        <>
          <Check className="h-4 w-4 text-green-600" />
          Copied!
        </>
      ) : (
        <>
          <Share2 className="h-4 w-4" />
          Share
        </>
      )}
    </Button>
  )
}