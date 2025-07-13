import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { Button } from '../ui/button'
import { Input } from '../ui/input'
import { ImageIcon } from 'lucide-react'
import { useWorkflowApi } from '../../hooks/use-workflow-api'
import { useRefreshStore } from '../../stores/refresh-store'
import { createImageGenerationFlow } from '../../workflows/definitions/image-generation'
import { DEFAULT_IMAGE_DIMENSIONS } from '../../constants/workflow'
import * as builders from '../../workflows/builders'

export function ImageGenerationDemo() {
  const [imageWidth, setImageWidth] = React.useState(DEFAULT_IMAGE_DIMENSIONS.WIDTH)
  const [imageHeight, setImageHeight] = React.useState(DEFAULT_IMAGE_DIMENSIONS.HEIGHT)
  const [loading, setLoading] = React.useState(false)
  const { flowCreate, runCommand } = useWorkflowApi()
  const { refreshAll } = useRefreshStore()

  const generateImage = async () => {
    setLoading(true)
    try {
      const flow = createImageGenerationFlow()
      await flowCreate(flow)
      
      const cmd = builders.createRunCommand(
        flow,
        builders.mapValue({
          prompt: builders.stringValue('no text'),
          width: builders.numberValue(imageWidth),
          height: builders.numberValue(imageHeight)
        })
      )
      
      const result = await runCommand(cmd)
      console.log('Image generation result:', result)
      
      // Refresh tables to show new workflow state
      refreshAll()
    } catch (error) {
      console.error('Error generating image:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ImageIcon className="h-5 w-5" />
          Image Generation
        </CardTitle>
        <CardDescription>Generate and resize images using workflows</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex gap-2">
          <Input
            type="number"
            placeholder="Width"
            value={imageWidth}
            onChange={(e) => setImageWidth(parseInt(e.target.value) || DEFAULT_IMAGE_DIMENSIONS.WIDTH)}
          />
          <Input
            type="number"
            placeholder="Height"
            value={imageHeight}
            onChange={(e) => setImageHeight(parseInt(e.target.value) || DEFAULT_IMAGE_DIMENSIONS.HEIGHT)}
          />
        </div>
        <Button onClick={generateImage} disabled={loading} variant="secondary">
          Generate image
        </Button>
      </CardContent>
    </Card>
  )
}