#!/bin/bash
# Manual capture guide for Leonardo Seedance 2.5

cat << 'EOF'
================================================================================
MANUAL NETWORK CAPTURE GUIDE - Leonardo Seedance 2.5
================================================================================

Since the automated capture sanitized too much data, let's capture manually:

STEP 1: Open Chrome DevTools
-----------------------------
1. Open Chrome/Chromium browser
2. Go to: https://app.leonardo.ai/generate
3. Press F12 to open DevTools
4. Click the "Network" tab
5. Make sure "Preserve log" is checked


STEP 2: Filter for GraphQL
---------------------------
1. In the filter box, type: graphql
2. Clear existing requests (trash icon)


STEP 3: Generate a Video
-------------------------
1. Select "Seedance 2.5" model
2. Enter a prompt (e.g., "cat running in the park")
3. Click "Generate"
4. Wait for the video to complete


STEP 4: Find the Generation Request
------------------------------------
Look in the Network tab for:
- Name: "graphql"
- Method: POST
- URL: https://api.leonardo.ai/v1/graphql

Right-click on it → "Copy" → "Copy as cURL (bash)"

Paste the cURL command into a file: /home/ziyad/seedance_curl.txt


STEP 5: Alternative - Copy Request Payload
-------------------------------------------
Or click on the request → "Payload" tab → Copy the JSON

The payload will look like:
{
  "operationName": "CreateVideoGeneration",
  "variables": {
    "prompt": "cat running in the park",
    "modelId": "...",
    "width": 1280,
    "height": 720,
    ...
  },
  "query": "mutation CreateVideoGeneration..."
}

Save this to: /home/ziyad/seedance_payload.json


STEP 6: Check leo2api Source
-----------------------------
Let me also check if leo2api already has hints about the GraphQL structure...

EOF

echo ""
echo "Checking leo2api source for GraphQL patterns..."
echo ""

grep -r "graphql\|GraphQL" ~/Documents/artemisia/leo2api-source/ --include="*.go" 2>/dev/null | head -20
