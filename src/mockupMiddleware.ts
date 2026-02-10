import type { Plugin } from "vite";
import { getMockData } from "./mockupData";
import { IncomingMessage, ServerResponse } from "http";

export function mockApiPlugin(): Plugin {
  return {
    name: "mock-api-middleware",
    configureServer(server) {
      server.middlewares.use(
        async (req: IncomingMessage, res: ServerResponse, next) => {
          // Only handle /api routes
          if (req.url?.startsWith("/api")) {
            try {
              const url = new URL(req.url, "http://localhost");
              const data = getMockData(url.pathname);

              if (!data) {
                res.statusCode = 404;
                res.end();
                return;
              }

              res.setHeader("Content-Type", "application/json");
              res.end(JSON.stringify(data));
            } catch (error) {
              console.error("Error in mock API middleware:", error);
              res.statusCode = 500;
              res.end(JSON.stringify({ error: "Internal Server Error" }));
            }
          } else {
            // Not an API request, continue to next middleware
            next();
          }
        }
      );
    },
  };
}
