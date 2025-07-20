import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../ui/card'
import { MessageSquareIcon } from 'lucide-react'
import { Chat } from '../../Chat'
import { useRefreshStore } from '../../stores/refresh-store'
import * as openai from '../../workflow/github_com_sashabaranov_go-openai'

export function ChatDemo() {
  const { refreshAll } = useRefreshStore()

  const handleFunctionCall = (func: openai.FunctionCall) => {
    
    // Refresh tables based on the function called
    if (func.Name === "refresh_flows" || func.Name === "refresh_states") {
      refreshAll()
    } else if (func.Name === "generate_image") {
      // Image generation will create a new workflow state
      refreshAll()
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <MessageSquareIcon className="h-5 w-5" />
          Chat Interface
        </CardTitle>
        <CardDescription>Interactive chat with function calling</CardDescription>
      </CardHeader>
      <CardContent>
        <Chat
          props={{
            name: "John",
            onFunctionCall: handleFunctionCall
          }}
        />
      </CardContent>
    </Card>
  )
}