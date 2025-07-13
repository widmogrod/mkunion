import React, { useState, useRef, useEffect } from 'react'
import ReactDOM from 'react-dom'
import { cn } from '../../lib/utils'

interface ResultPreviewProps {
  result: any
  className?: string
  maxTextLength?: number
  thumbnailSize?: number
  previewSize?: number
}

export function ResultPreview({ 
  result, 
  className,
  maxTextLength = 60,
  thumbnailSize = 32,
  previewSize = 200 
}: ResultPreviewProps) {
  const [showPreview, setShowPreview] = useState(false)
  const [showFullText, setShowFullText] = useState(false)
  const [imageError, setImageError] = useState(false)
  const [imageLoaded, setImageLoaded] = useState(false)
  const [isVisible, setIsVisible] = useState(false)
  const [mousePosition, setMousePosition] = useState({ x: 0, y: 0 })
  const previewRef = useRef<HTMLDivElement>(null)
  const hoverTimeoutRef = useRef<NodeJS.Timeout>()
  const observerRef = useRef<IntersectionObserver>()
  const imageRef = useRef<HTMLDivElement>(null)
  const thumbnailRef = useRef<HTMLImageElement>(null)

  // Set up intersection observer for lazy loading
  useEffect(() => {
    if (!imageRef.current || imageLoaded) return

    observerRef.current = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            setIsVisible(true)
            observerRef.current?.disconnect()
          }
        })
      },
      { 
        rootMargin: '50px', // Start loading 50px before image enters viewport
        threshold: 0.01 
      }
    )

    observerRef.current.observe(imageRef.current)

    return () => {
      observerRef.current?.disconnect()
    }
  }, [imageLoaded])

  // Clean up hover timeout on unmount
  useEffect(() => {
    return () => {
      if (hoverTimeoutRef.current) {
        clearTimeout(hoverTimeoutRef.current)
      }
    }
  }, [])

  const handleMouseEnter = (e: React.MouseEvent) => {
    // Get position of the thumbnail
    if (thumbnailRef.current) {
      const rect = thumbnailRef.current.getBoundingClientRect()
      setMousePosition({
        x: rect.left + rect.width / 2,
        y: rect.bottom + 8
      })
    }
    
    // Debounce hover to prevent flickering
    hoverTimeoutRef.current = setTimeout(() => {
      setShowPreview(true)
    }, 200) // Reduced delay for better responsiveness
  }

  const handleMouseLeave = () => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
    }
    setShowPreview(false)
  }

  if (!result) {
    return <span className="text-xs text-muted-foreground">No result</span>
  }

  // Handle schema.Binary type
  if (result.$type === 'schema.Binary' && result['schema.Binary']) {
    const base64Data = result['schema.Binary']
    
    // Detect MIME type from base64 signature
    let mimeType: string | null = null
    let isImage = false
    
    // Check for image signatures
    if (base64Data.startsWith('iVBORw0KGgo')) {
      mimeType = 'image/png'
      isImage = true
    } else if (base64Data.startsWith('R0lGOD')) {
      mimeType = 'image/gif'
      isImage = true
    } else if (base64Data.startsWith('Qk')) {
      mimeType = 'image/bmp'
      isImage = true
    } else if (base64Data.startsWith('UklGR')) {
      mimeType = 'image/webp'
      isImage = true
    } else if (base64Data.startsWith('/9j/') || base64Data.startsWith('/9J/')) {
      mimeType = 'image/jpeg'
      isImage = true
    } else if (base64Data.startsWith('JVBERi')) {
      mimeType = 'application/pdf'
    } else if (base64Data.startsWith('UEsDBA')) {
      mimeType = 'application/zip'
    } else if (base64Data.startsWith('e1x')) {
      mimeType = 'application/rtf'
    } else {
      // Try to detect text
      try {
        const decoded = atob(base64Data.slice(0, 100))
        if (/^[\x20-\x7E\r\n\t]+$/.test(decoded)) {
          mimeType = 'text/plain'
        }
      } catch {
        // Not valid base64 or binary data
      }
    }

    // Default to JPEG for unknown image-like data
    if (!mimeType && !imageError) {
      mimeType = 'image/jpeg'
      isImage = true
    }

    if (isImage && mimeType) {
      const imageSrc = `data:${mimeType};base64,${base64Data}`

      return (
        <div className={cn("inline-flex items-center gap-1", className)} style={{ position: 'relative' }}>
          <span className="text-xs text-muted-foreground">schema.Binary:</span>
          <div 
            ref={imageRef}
            className="relative inline-block"
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            style={{ position: 'relative' }}
          >
            {/* Thumbnail with lazy loading */}
            {isVisible ? (
              <img
                ref={thumbnailRef}
                src={imageSrc}
                alt="Binary result"
                className={cn(
                  "rounded border border-border cursor-pointer transition-transform hover:scale-105",
                  imageError && "bg-muted",
                  !imageLoaded && "animate-pulse bg-muted"
                )}
                style={{ 
                  width: thumbnailSize, 
                  height: thumbnailSize,
                  objectFit: 'cover'
                }}
                onLoad={() => setImageLoaded(true)}
                onError={(e) => {
                  setImageError(true)
                  setImageLoaded(true)
                  // Try fallback to JPEG
                  const img = e.target as HTMLImageElement
                  if (img.src !== `data:image/jpeg;base64,${base64Data}`) {
                    img.src = `data:image/jpeg;base64,${base64Data}`
                  }
                }}
                onClick={(e) => {
                  e.stopPropagation()
                  // TODO: Open in lightbox/modal
                  window.open(imageSrc, '_blank')
                }}
              />
            ) : (
              // Placeholder while loading
              <div 
                className="rounded border border-border bg-muted animate-pulse"
                style={{ 
                  width: thumbnailSize, 
                  height: thumbnailSize 
                }}
              />
            )}
            
            {/* Hover Preview Portal - renders outside overflow container */}
            {showPreview && !imageError && imageLoaded && ReactDOM.createPortal(
              <div 
                ref={previewRef}
                className="fixed z-[1000] p-2 bg-background border-2 border-border rounded-lg shadow-xl"
                style={{
                  top: mousePosition.y,
                  left: mousePosition.x,
                  transform: 'translateX(-50%)',
                  minWidth: previewSize,
                  pointerEvents: 'none' // Prevent hover interference
                }}
              >
                <img
                  src={imageSrc}
                  alt="Binary preview expanded"
                  className="rounded block"
                  style={{ 
                    maxWidth: previewSize, 
                    maxHeight: previewSize,
                    width: 'auto',
                    height: 'auto'
                  }}
                />
                <div className="mt-1 text-xs text-muted-foreground text-center">
                  {mimeType}
                </div>
              </div>,
              document.body
            )}
            
            {imageError && (
              <span className="text-xs text-muted-foreground">🖼️ {mimeType}</span>
            )}
          </div>
        </div>
      )
    } else {
      // Non-image binary data
      const preview = base64Data.slice(0, maxTextLength)
      const sizeEstimate = Math.round(base64Data.length * 0.75) // Rough estimate of decoded size
      const sizeStr = sizeEstimate > 1024 ? `${Math.round(sizeEstimate / 1024)}KB` : `${sizeEstimate}B`
      
      return (
        <div className={cn("inline text-xs", className)}>
          <span className="text-muted-foreground">schema.Binary</span>
          {mimeType && <span className="text-muted-foreground ml-1">({mimeType})</span>}
          <span className="text-muted-foreground ml-1">[{sizeStr}]:</span>
          <span className="ml-1 font-mono text-muted-foreground">{preview}</span>
          {base64Data.length > maxTextLength && (
            <>
              <span className="text-muted-foreground">...</span>
              <button
                onClick={(e) => {
                  e.stopPropagation()
                  setShowFullText(!showFullText)
                }}
                className="ml-1 text-blue-600 hover:text-blue-700 underline"
              >
                {showFullText ? 'less' : 'more'}
              </button>
            </>
          )}
          {showFullText && (
            <div className="mt-1 font-mono text-xs bg-muted/20 p-2 rounded overflow-x-auto max-w-full">
              {base64Data}
            </div>
          )}
        </div>
      )
    }
  }

  // Handle string results
  if (result.$type === 'schema.String' && result['schema.String'] !== undefined) {
    const text = String(result['schema.String'])
    const isLong = text.length > maxTextLength
    const displayText = showFullText ? text : text.slice(0, maxTextLength)

    return (
      <div className={cn("inline text-xs", className)}>
        <span className="text-muted-foreground">schema.String:</span>
        <span className="ml-1 text-green-600">"{displayText}"</span>
        {isLong && !showFullText && (
          <>
            <span className="text-muted-foreground">...</span>
            <button
              onClick={(e) => {
                e.stopPropagation()
                setShowFullText(true)
              }}
              className="ml-1 text-blue-600 hover:text-blue-700 underline"
            >
              more
            </button>
          </>
        )}
        {isLong && showFullText && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setShowFullText(false)
            }}
            className="ml-1 text-blue-600 hover:text-blue-700 underline"
          >
            less
          </button>
        )}
      </div>
    )
  }

  // Handle number results
  if (result.$type === 'schema.Number' && result['schema.Number'] !== undefined) {
    return (
      <span className={cn("text-xs", className)}>
        <span className="text-muted-foreground">schema.Number:</span>
        <span className="ml-1 text-blue-600">{result['schema.Number']}</span>
      </span>
    )
  }

  // Handle boolean results
  if (result.$type === 'schema.Bool' && result['schema.Bool'] !== undefined) {
    return (
      <span className={cn("text-xs", className)}>
        <span className="text-muted-foreground">schema.Bool:</span>
        <span className="ml-1 text-purple-600">{result['schema.Bool'] ? 'true' : 'false'}</span>
      </span>
    )
  }

  // Handle Map and complex objects
  if (result.$type === 'schema.Map' || (result.$type && typeof result[result.$type] === 'object')) {
    const data = result[result.$type] || result
    const keys = Object.keys(data)
    const preview = `{${result.$type}: ${keys.length} ${keys.length === 1 ? 'property' : 'properties'}}`
    
    return (
      <div className={cn("inline", className)}>
        <span className="text-xs text-muted-foreground">{preview}</span>
        <button
          onClick={(e) => {
            e.stopPropagation()
            // TODO: Implement expand view
            console.log('Expand complex object:', data)
          }}
          className="ml-1 text-xs text-blue-600 hover:text-blue-700 underline"
        >
          view
        </button>
      </div>
    )
  }

  // Handle arrays
  if (Array.isArray(result)) {
    return (
      <span className={cn("text-xs text-muted-foreground", className)}>
        [Array: {result.length} items]
      </span>
    )
  }

  // Fallback for unknown types
  const resultStr = JSON.stringify(result)
  const isLongJson = resultStr.length > maxTextLength
  const displayJson = showFullText ? resultStr : resultStr.slice(0, maxTextLength)

  return (
    <div className={cn("inline text-xs", className)}>
      <span className="text-muted-foreground font-mono">{displayJson}</span>
      {isLongJson && !showFullText && (
        <>
          <span className="text-muted-foreground">...</span>
          <button
            onClick={(e) => {
              e.stopPropagation()
              setShowFullText(true)
            }}
            className="ml-1 text-blue-600 hover:text-blue-700 underline"
          >
            more
          </button>
        </>
      )}
      {isLongJson && showFullText && (
        <button
          onClick={(e) => {
            e.stopPropagation()
            setShowFullText(false)
          }}
          className="ml-1 text-blue-600 hover:text-blue-700 underline"
        >
          less
        </button>
      )}
    </div>
  )
}