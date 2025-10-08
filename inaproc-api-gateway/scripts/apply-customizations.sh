#!/bin/bash
# LKPP Customization Script for Fusio Apps
# This script applies LKPP branding (UI/visual changes only) after marketplace installation
# Usage: ./scripts/apply-customizations.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CUSTOMIZATIONS_DIR="$PROJECT_ROOT/customizations"
PUBLIC_APPS_DIR="$PROJECT_ROOT/public/apps"

echo "=================================================="
echo "  LKPP Fusio Customization Script"
echo "  (UI/Visual Changes Only)"
echo "=================================================="
echo ""

# Check if customizations directory exists
if [ ! -d "$CUSTOMIZATIONS_DIR" ]; then
    echo "❌ Error: Customizations directory not found: $CUSTOMIZATIONS_DIR"
    exit 1
fi

# Function to apply account app visual customizations
apply_account_customizations() {
    echo "📦 Applying Account App Visual Customizations..."
    
    ACCOUNT_APP_DIR="$PUBLIC_APPS_DIR/account"
    
    if [ ! -d "$ACCOUNT_APP_DIR" ]; then
        echo "⚠️  Warning: Account app not found at $ACCOUNT_APP_DIR"
        echo "   Please run: php bin/fusio marketplace:install account"
        return 1
    fi
    
    # 1. Copy custom assets and replace Fusio default logos
    echo "  → Copying custom assets..."
    if [ -f "$CUSTOMIZATIONS_DIR/account/assets/apigw_favicon.png" ]; then
        cp "$CUSTOMIZATIONS_DIR/account/assets/apigw_favicon.png" "$ACCOUNT_APP_DIR/assets/"
        echo "    ✓ Copied apigw_favicon.png"
    fi

    # Replace Fusio default logos with LKPP logo
    if [ -f "$CUSTOMIZATIONS_DIR/account/assets/lkpp-logo.png" ]; then
        # Copy as lkpp-logo.png (for any code that references it)
        cp "$CUSTOMIZATIONS_DIR/account/assets/lkpp-logo.png" "$ACCOUNT_APP_DIR/assets/"
        echo "    ✓ Copied lkpp-logo.png"
        
        # Replace fusio_64px.png with LKPP logo
        cp "$CUSTOMIZATIONS_DIR/account/assets/lkpp-logo.png" "$ACCOUNT_APP_DIR/assets/fusio_64px.png"
        echo "    ✓ Replaced fusio_64px.png with LKPP logo"
        
        # Replace fusio_32px.png with LKPP logo
        cp "$CUSTOMIZATIONS_DIR/account/assets/lkpp-logo.png" "$ACCOUNT_APP_DIR/assets/fusio_32px.png"
        echo "    ✓ Replaced fusio_32px.png with LKPP logo"
    fi

    if [ -f "$CUSTOMIZATIONS_DIR/account/assets/lkpp-logo1.png" ]; then
        cp "$CUSTOMIZATIONS_DIR/account/assets/lkpp-logo1.png" "$ACCOUNT_APP_DIR/assets/"
        echo "    ✓ Copied lkpp-logo1.png"
    fi
    
    # 2. Copy custom theme CSS
    echo "  → Copying LKPP theme CSS..."
    if [ -f "$CUSTOMIZATIONS_DIR/account/lkpp-theme.css" ]; then
        cp "$CUSTOMIZATIONS_DIR/account/lkpp-theme.css" "$ACCOUNT_APP_DIR/"
        echo "    ✓ Copied lkpp-theme.css"
    fi
    
    # 3. Patch index.html (ONLY UI changes, no functional changes)
    echo "  → Patching index.html for visual changes..."
    
    INDEX_FILE="$ACCOUNT_APP_DIR/index.html"
    
    if [ ! -f "$INDEX_FILE" ]; then
        echo "❌ Error: index.html not found"
        return 1
    fi
    
    # Change favicon to custom LKPP favicon
    sed -i.bak 's|href="assets/fusio_32px.png"|href="assets/apigw_favicon.png"|g' "$INDEX_FILE"
    sed -i.bak 's|type="image/x-icon"|type="image/png"|g' "$INDEX_FILE"
    
    # Add Google Fonts preconnect (if not already present) for custom fonts
    if ! grep -q "fonts.googleapis.com" "$INDEX_FILE"; then
        sed -i.bak '/<meta name="viewport"/a\
  <link rel="preconnect" href="https://fonts.googleapis.com">\
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>' "$INDEX_FILE"
    fi
    
    # Add lkpp-theme.css link before </head> (if not already present)
    if ! grep -q "lkpp-theme.css" "$INDEX_FILE"; then
        sed -i.bak 's|</head>|  <link rel="stylesheet" href="lkpp-theme.css">\n</head>|g' "$INDEX_FILE"
    fi
    
    # Inject branding customization script before </head> (if not already present)
    if ! grep -q "brandCaption" "$INDEX_FILE" && [ -f "$CUSTOMIZATIONS_DIR/patches/account-branding.js" ]; then
        BRANDING_SCRIPT=$(cat "$CUSTOMIZATIONS_DIR/patches/account-branding.js")
        # Create a temporary file with the script wrapped in <script> tags
        echo "  <script>" > /tmp/branding-inject.txt
        echo "$BRANDING_SCRIPT" >> /tmp/branding-inject.txt
        echo "  </script>" >> /tmp/branding-inject.txt
        
        # Insert before lkpp-theme.css link
        sed -i.bak -e "/<link rel=\"stylesheet\" href=\"lkpp-theme.css\">/r /tmp/branding-inject.txt" "$INDEX_FILE"
        rm /tmp/branding-inject.txt
        echo "    ✓ Added branding script"
    fi
    
    # Add lkpp-theme class to body tag (if not already present)
    if ! grep -q 'class="lkpp-theme"' "$INDEX_FILE"; then
        sed -i.bak 's|<body>|<body class="lkpp-theme">|g' "$INDEX_FILE"
    fi
    
    # Clean up backup files
    rm -f "$INDEX_FILE.bak"
    
    echo "✅ Account app visual customizations applied successfully!"
    echo ""
}

# Main execution
main() {
    echo "Starting customization process..."
    echo ""
    
    # Apply account app customizations
    if apply_account_customizations; then
        echo "✅ Account app customizations completed"
    else
        echo "⚠️  Account app customizations had issues"
    fi
    
    echo ""
    echo "=================================================="
    echo "  ✅ Customization Complete!"
    echo "=================================================="
    echo ""
    echo "Summary of changes:"
    echo "  • Replaced Fusio logo files with LKPP logo"
    echo "  • Applied LKPP theme CSS (colors, fonts, styling)"
    echo "  • Updated favicon to LKPP branding"
    echo "  • Added INAPROC branding caption"
    echo ""
    echo "⚠️  Note: No functional changes were made"
    echo "   (URLs, API endpoints, and app functionality remain unchanged)"
    echo ""
    echo "Next steps:"
    echo "1. Test the UI changes at http://localhost:8080/apps/account/login"
    echo "2. If everything looks good, commit the changes"
    echo "3. For production: rebuild Docker image and deploy"
    echo ""
}

# Run main function
main
