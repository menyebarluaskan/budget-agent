# Intergated Personal Budget

This is an budgeting tool that will run in a containerized server
with its own pod and sidecar container that using tailscale. It will
having an HTTP API for manipulating its data.

## Main Components

1. Optimized backends with low memory usage and small footprint database
2. AI Agents or MCP? to access the API, and how to connect them with whatsapp
   chat bot
3. Web Front End that mobile friendly for tracking transaction

## Core Features

1. Budgeting Features
   1. Add Transactions with multiple tags and simple plus(+) or minus (-) as
      expense indicator
   2. Budgets that are grouped by multiple tags from transaction
   3. Wallets, savings, and credit accounts
   4. Data filtering based on accounts, tags, budgets, and transaction amounts
   5. Budget summary and spending recommendations

2. AI Agents Features
   1. OCR from receipt and scan its spending and summarize the description
   2. Add transaction by chat message

3. Web Front End Features
   1. Budgeting Dashboard
   2. Spending graph and its simulation
