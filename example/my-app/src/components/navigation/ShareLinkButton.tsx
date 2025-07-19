import React, { useState } from 'react'
import { Share2, Check, Copy } from 'lucide-react'
import { Button } from '../ui/button'
import { useShareableLink } from '../../hooks/useNavigation'
import { useToast } from '../../contexts/ToastContext'

export function ShareLinkButton() {
  const { shareableLink, copyToClipboard } = useShareableLink()
  const [copied, setCopied] = useState(false)
  const toast = useToast()

  const handleShare = async () => {
    const success = await copyToClipboard()
    if (success) {
      setCopied(true)
      toast.success('Link Copied!', 'The shareable link has been copied to your clipboard')
      setTimeout(() => setCopied(false), 2000)
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