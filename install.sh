#!/bin/bash
# K8s Operator Skills - One-Click Installation Script

set -e

REPO_URL="https://github.com/hawkli-1994/k8s-operator-skills.git"
INSTALL_DIR="$HOME/.claude/skills/k8s-operator"

echo "🚀 Installing K8s Operator Development Skill..."
echo ""

# Check if directory already exists
if [ -d "$INSTALL_DIR" ]; then
    echo "⚠️  Skill already installed at $INSTALL_DIR"
    read -p "Update to latest version? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "📥 Updating..."
        cd "$INSTALL_DIR"
        git pull origin main
        echo "✅ Skill updated successfully!"
    else
        echo "ℹ️  Installation skipped. Skill is already installed."
    fi
else
    echo "📥 Cloning repository..."
    git clone "$REPO_URL" "$INSTALL_DIR"
    echo "✅ Skill installed successfully!"
fi

echo ""
echo "🎉 Installation complete!"
echo ""
echo "The K8s Operator Development Skill is now available."
echo ""
echo "Quick start prompts to try:"
echo '  • "Help me create a Kubernetes operator for managing databases"'
echo '  • "Show me the advanced reconciler patterns"'
echo '  • "How do I set up CI/CD for my operator?"'
echo '  • "Write tests for my reconciler"'
echo ""
echo "📚 Documentation: https://github.com/hawkli-1994/k8s-operator-skills"
