# n8n Workflow Docker Configuration
# Based on official n8n Docker image with custom workflow setup
FROM docker.n8n.io/n8nio/n8n:1.113.3

# Set working directory
WORKDIR /home/node

# Create necessary directories
USER root
RUN mkdir -p /home/node/.n8n/workflows \
    && mkdir -p /home/node/.n8n/credentials \
    && mkdir -p /home/node/workflows-backup \
    && chown -R node:node /home/node

# Switch back to node user
USER node

# Copy workflow files
COPY --chown=node:node n8n/workflows/ /home/node/.n8n/workflows/
COPY --chown=node:node docs/ /home/node/workflows-backup/docs/

# Environment variables for production (will be overridden by k8s)
ENV NODE_ENV=production
ENV N8N_HOST=0.0.0.0
ENV N8N_PORT=5678
ENV N8N_PROTOCOL=https
ENV GENERIC_TIMEZONE=Asia/Jakarta
ENV N8N_METRICS=true
ENV N8N_LOG_LEVEL=info
ENV N8N_LOG_OUTPUT=console
ENV EXECUTIONS_PROCESS=main
ENV EXECUTIONS_DATA_SAVE_ON_ERROR=all
ENV EXECUTIONS_DATA_SAVE_ON_SUCCESS=all

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD curl -f http://localhost:5678/healthz || exit 1

# Expose port
EXPOSE 5678

# Start n8n
CMD ["n8n", "start"]