#!/bin/bash

# Change to Fusio directory
pushd /var/www/html/fusio

# Function to wait for database to be ready
wait_for_db() {
    echo "Waiting for database to be ready..."
    until php -r "
        try {
            \$connection = getenv('FUSIO_CONNECTION') ?: getenv('APP_CONNECTION');
            if (strpos(\$connection, 'pdo-pgsql://') === 0) {
                \$url = parse_url(str_replace('pdo-pgsql://', 'postgres://', \$connection));
                \$pdo = new PDO('pgsql:host='.\$url['host'].';port='.(\$url['port'] ?? 5432).';dbname='.ltrim(\$url['path'], '/'), \$url['user'], \$url['pass']);
                echo 'Database connection successful' . PHP_EOL;
                exit(0);
            }
            echo 'Unsupported connection format' . PHP_EOL;
            exit(1);
        } catch (Exception \$e) {
            echo 'Database connection failed: ' . \$e->getMessage() . PHP_EOL;
            exit(1);
        }
    "; do
        echo "Database not ready, waiting 3 seconds..."
        sleep 3
    done
}

echo "Starting Fusio Apache initialization..."

# Update .env file with environment variables from deployment dynamically
echo "Updating .env file with deployment environment variables..."
ENV_FILE="/var/www/html/fusio/.env"

if [ -f "$ENV_FILE" ]; then
    echo "Reading .env file and updating with container environment variables..."
    
    # Create a temporary file for the updated .env
    temp_env="/tmp/.env.tmp"
    cp "$ENV_FILE" "$temp_env"
    
    # Read each line from .env file
    while IFS= read -r line || [ -n "$line" ]; do
        # Skip empty lines and comments (lines starting with #)
        if [ -z "$line" ] || echo "$line" | grep -q '^[[:space:]]*#'; then
            continue
        fi
        
        # Extract variable name (everything before the first =)
        var_name=$(echo "$line" | cut -d'=' -f1 | tr -d '[:space:]')
        
        # Skip if no variable name found
        if [ -z "$var_name" ]; then
            continue
        fi
        
        # Check if this variable exists in container environment
        env_value=$(printenv "$var_name")
        
        if [ -n "$env_value" ]; then
            echo "  Updating $var_name with container environment value"
            # Replace the line in temp file
            sed -i "s|^[[:space:]]*$var_name=.*|$var_name=\"$env_value\"|" "$temp_env"
        else
            echo "  Container environment variable $var_name not found, keeping default value"
        fi
        
    done < "$ENV_FILE"
    
    # Replace original .env with updated version
    mv "$temp_env" "$ENV_FILE"
    
else
    echo "Warning: .env file not found at $ENV_FILE"
fi

echo "Updated .env file contents:"
cat "$ENV_FILE"

# Set environment variables from local env if they exist
if [ -f "/var/www/html/fusio/.env.local" ]; then
    echo "Loading .env.local configuration..."
    export $(grep -v '^#' /var/www/html/fusio/.env.local | xargs)
fi

# Override environment variables for consistency
export FUSIO_PROJECT_KEY="${APP_PROJECT_KEY:-$FUSIO_PROJECT_KEY}"
export FUSIO_ENV="${APP_ENV:-$FUSIO_ENV}"
export FUSIO_DEBUG="${APP_DEBUG:-$FUSIO_DEBUG}"
export FUSIO_CONNECTION="${APP_CONNECTION:-$FUSIO_CONNECTION}"
export FUSIO_MAILER="${APP_MAILER:-$FUSIO_MAILER}"
export FUSIO_MESSENGER="${APP_MESSENGER:-$FUSIO_MESSENGER}"
export FUSIO_URL="${APP_URL:-$FUSIO_URL}"
export FUSIO_APPS_URL="${APP_APPS_URL:-$FUSIO_APPS_URL}"

echo "Configuration:"
echo "  PROJECT_KEY: $FUSIO_PROJECT_KEY"
echo "  ENV: $FUSIO_ENV"
echo "  DEBUG: $FUSIO_DEBUG"
echo "  CONNECTION: $FUSIO_CONNECTION"
echo "  URL: $FUSIO_URL"
echo "  APPS_URL: $FUSIO_APPS_URL"

# Wait for database to be available
wait_for_db

# Wait for external services (using Fusio's built-in wait command)
echo "Waiting for external services..."
php bin/fusio system:wait_for

# Run Fusio migration
echo "Running Fusio migration..."
php bin/fusio migration:up-to-date
if [ $? -ne 0 ]; then
    echo "Running initial migration..."
    php bin/fusio migration:migrate --no-interaction
fi

# Check if backend user exists and create if needed
echo "Checking backend user..."
php bin/fusio system:check user
if [ $? -ne 0 ]; then
    echo "Creating backend user..."
    php bin/fusio adduser --role=1 --username="$FUSIO_BACKEND_USER" --email="$FUSIO_BACKEND_EMAIL" --password="$FUSIO_BACKEND_PW"
fi

# Replace environment variables in apps
# echo "Updating app configurations..."
# php bin/fusio marketplace:env -

# Install backend apps if they don't exist
if [ ! -d "/var/www/html/fusio/public/apps/fusio" ]; then
    echo "Installing Fusio backend app..."
    php bin/fusio marketplace:install fusio || echo "Backend app installation failed, continuing..."
fi

# Deploy any pending changes
echo "Deploying changes..."
php bin/fusio deploy

echo "Fusio initialization completed successfully!"

# Start cron service
echo "Starting cron service..."
service cron start

# Start Apache in the foreground
echo "Starting Apache..."
exec apache2-foreground