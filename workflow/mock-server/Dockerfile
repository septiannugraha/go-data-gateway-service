FROM node:18-alpine

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy server code
COPY server.js ./

# Expose port
EXPOSE 8000

# Run the server
CMD ["node", "server.js"]