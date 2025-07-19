import React from 'react'
import { AlertCircle } from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '../../../ui/alert'
import { Button } from '../../../ui/button'

interface ErrorStateProps {
  error: Error
  onRetry?: () => void
  className?: string
}

export function ErrorState({ 
  error, 
  onRetry,
  className = ""
}: ErrorStateProps) {
  return (
    <div className={`p-4 ${className}`}>
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Error loading data</AlertTitle>
        <AlertDescription className="mt-2">
          <p>{error.message || 'An unexpected error occurred'}</p>
          {onRetry && (
            <Button 
              variant="outline" 
              size="sm" 
              onClick={onRetry}
              className="mt-4"
            >
              Try again
            </Button>
          )}
        </AlertDescription>
      </Alert>
    </div>
  )
}