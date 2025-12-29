#!/bin/bash

# DB-Taxi Debug Script
echo "🔍 DB-Taxi Debug Information"
echo "============================"
echo ""

# Check if binary exists
if [ ! -f "./db-taxi" ]; then
    echo "❌ db-taxi binary not found. Please build first:"
    echo "   go build -o db-taxi ."
    exit 1
fi

# Check static files
echo "📁 Static Files Check"
echo "---------------------"
if [ -d "./static" ]; then
    echo "✅ static/ directory exists"
    if [ -f "./static/index.html" ]; then
        echo "✅ static/index.html exists"
        echo "   Size: $(wc -c < ./static/index.html) bytes"
    else
        echo "❌ static/index.html missing"
    fi
else
    echo "❌ static/ directory missing"
fi
echo ""

# Check configuration
echo "⚙️  Configuration Check"
echo "----------------------"
if [ -f "./config.yaml" ]; then
    echo "✅ config.yaml exists"
    echo "   Server port: $(grep -A5 "^server:" config.yaml | grep "port:" | awk '{print $2}')"
    echo "   Database host: $(grep -A10 "^database:" config.yaml | grep "host:" | awk '{print $2}' | tr -d '"')"
else
    echo "ℹ️  config.yaml not found (will use defaults)"
fi
echo ""

# Check network connectivity
echo "🌐 Network Check"
echo "---------------"
PORT=${1:-8080}
echo "Checking if port $PORT is available..."

if command -v lsof &> /dev/null; then
    if lsof -i :$PORT &> /dev/null; then
        echo "⚠️  Port $PORT is already in use:"
        lsof -i :$PORT
    else
        echo "✅ Port $PORT is available"
    fi
elif command -v netstat &> /dev/null; then
    if netstat -ln | grep ":$PORT " &> /dev/null; then
        echo "⚠️  Port $PORT appears to be in use"
    else
        echo "✅ Port $PORT appears to be available"
    fi
else
    echo "ℹ️  Cannot check port availability (lsof/netstat not found)"
fi
echo ""

# Test database connection if MySQL client is available
echo "🗄️  Database Connection Check"
echo "-----------------------------"
if command -v mysql &> /dev/null; then
    if [ -f "./config.yaml" ]; then
        DB_HOST=$(grep -A10 "^database:" config.yaml | grep "host:" | awk '{print $2}' | tr -d '"')
        DB_PORT=$(grep -A10 "^database:" config.yaml | grep "port:" | awk '{print $2}')
        DB_USER=$(grep -A10 "^database:" config.yaml | grep "username:" | awk '{print $2}' | tr -d '"')
        
        echo "Testing connection to $DB_HOST:$DB_PORT as $DB_USER..."
        if mysql -h "$DB_HOST" -P "$DB_PORT" -u "$DB_USER" -e "SELECT 1;" 2>/dev/null; then
            echo "✅ Database connection successful"
        else
            echo "❌ Database connection failed"
            echo "   This is normal if you haven't set up MySQL yet"
            echo "   DB-Taxi will still work but show connection errors"
        fi
    else
        echo "ℹ️  No config file found, skipping database test"
    fi
else
    echo "ℹ️  MySQL client not found, skipping database test"
fi
echo ""

# Quick server test
echo "🚀 Quick Server Test"
echo "-------------------"
echo "Starting DB-Taxi for 5 seconds to test basic functionality..."
echo ""

# Start server in background
./db-taxi -server-port $PORT &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Test endpoints
echo "Testing endpoints:"

# Test health endpoint
if curl -s "http://localhost:$PORT/health" > /dev/null 2>&1; then
    echo "✅ Health endpoint: http://localhost:$PORT/health"
else
    echo "❌ Health endpoint failed"
fi

# Test root endpoint
if curl -s "http://localhost:$PORT/" > /dev/null 2>&1; then
    echo "✅ Root endpoint: http://localhost:$PORT/"
else
    echo "❌ Root endpoint failed"
fi

# Test API status
if curl -s "http://localhost:$PORT/api/status" > /dev/null 2>&1; then
    echo "✅ API status: http://localhost:$PORT/api/status"
else
    echo "❌ API status failed"
fi

# Stop server
sleep 1
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null

echo ""
echo "🎉 Debug check completed!"
echo ""
echo "💡 Common Issues & Solutions:"
echo "   • 404 errors: Make sure static/index.html exists"
echo "   • Port conflicts: Use -server-port to change port"
echo "   • Database errors: Check MySQL connection settings"
echo "   • Permission errors: Make sure db-taxi binary is executable"
echo ""
echo "🔗 If everything looks good, start DB-Taxi with:"
echo "   ./db-taxi"
echo "   Then visit: http://localhost:$PORT"