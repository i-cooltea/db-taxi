#!/bin/bash

echo "🚕 DB-Taxi Frontend Development Server"
echo "======================================"
echo ""

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
    echo ""
fi

echo "🚀 Starting development server..."
echo "Frontend: http://localhost:3000"
echo "API Proxy: http://localhost:8080"
echo ""
echo "Press Ctrl+C to stop"
echo ""

npm run dev
