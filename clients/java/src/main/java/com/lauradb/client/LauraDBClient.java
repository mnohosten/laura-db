package com.lauradb.client;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.CompletableFuture;

/**
 * LauraDB Java Client for interacting with LauraDB HTTP server.
 *
 * <p>Example usage:
 * <pre>{@code
 * LauraDBClient client = LauraDBClient.builder()
 *     .host("localhost")
 *     .port(8080)
 *     .build();
 *
 * boolean healthy = client.ping();
 * Collection users = client.collection("users");
 * }</pre>
 */
public class LauraDBClient implements AutoCloseable {
    private final String host;
    private final int port;
    private final boolean https;
    private final int timeout;
    private final String baseUrl;
    private final Gson gson;

    private LauraDBClient(Builder builder) {
        this.host = builder.host;
        this.port = builder.port;
        this.https = builder.https;
        this.timeout = builder.timeout;

        String protocol = https ? "https" : "http";
        this.baseUrl = String.format("%s://%s:%d", protocol, host, port);
        this.gson = new Gson();
    }

    /**
     * Creates a new builder for configuring the client.
     *
     * @return a new Builder instance
     */
    public static Builder builder() {
        return new Builder();
    }

    /**
     * Builder for creating LauraDBClient instances.
     */
    public static class Builder {
        private String host = "localhost";
        private int port = 8080;
        private boolean https = false;
        private int timeout = 30000; // 30 seconds in milliseconds

        public Builder host(String host) {
            this.host = host;
            return this;
        }

        public Builder port(int port) {
            this.port = port;
            return this;
        }

        public Builder https(boolean https) {
            this.https = https;
            return this;
        }

        public Builder timeout(int timeout) {
            this.timeout = timeout;
            return this;
        }

        public LauraDBClient build() {
            return new LauraDBClient(this);
        }
    }

    /**
     * Get a collection by name.
     *
     * @param name the collection name
     * @return a Collection instance
     */
    public Collection collection(String name) {
        return new Collection(this, name);
    }

    /**
     * Ping the server to check if it's alive.
     *
     * @return true if the server is healthy, false otherwise
     */
    public boolean ping() {
        try {
            JsonObject response = request("GET", "/_health", null);
            return response.has("ok") && response.get("ok").getAsBoolean();
        } catch (IOException e) {
            return false;
        }
    }

    /**
     * Get database statistics.
     *
     * @return statistics as a map
     * @throws IOException if the request fails
     */
    public Map<String, Object> stats() throws IOException {
        JsonObject response = request("GET", "/_stats", null);
        return gson.fromJson(response, Map.class);
    }

    /**
     * List all collections.
     *
     * @return list of collection names
     * @throws IOException if the request fails
     */
    @SuppressWarnings("unchecked")
    public List<String> listCollections() throws IOException {
        JsonObject response = request("GET", "/_collections", null);
        return gson.fromJson(response.get("collections"), List.class);
    }

    /**
     * Create a new collection.
     *
     * @param name the collection name
     * @throws IOException if the request fails
     */
    public void createCollection(String name) throws IOException {
        Map<String, Object> body = new HashMap<>();
        body.put("name", name);
        request("POST", "/_collections", body);
    }

    /**
     * Drop a collection.
     *
     * @param name the collection name
     * @throws IOException if the request fails
     */
    public void dropCollection(String name) throws IOException {
        request("DELETE", "/" + name, null);
    }

    /**
     * Perform an asynchronous ping.
     *
     * @return a CompletableFuture that completes with true if healthy
     */
    public CompletableFuture<Boolean> pingAsync() {
        return CompletableFuture.supplyAsync(this::ping);
    }

    /**
     * Perform an HTTP request to the LauraDB server.
     *
     * @param method HTTP method (GET, POST, PUT, DELETE)
     * @param path request path (relative to base URL)
     * @param body request body (will be JSON encoded, can be null)
     * @return response as JsonObject
     * @throws IOException if the HTTP request fails
     */
    JsonObject request(String method, String path, Object body) throws IOException {
        URL url = new URL(baseUrl + path);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();

        try {
            conn.setRequestMethod(method);
            conn.setConnectTimeout(timeout);
            conn.setReadTimeout(timeout);
            conn.setRequestProperty("Accept", "application/json");
            conn.setRequestProperty("User-Agent", "LauraDB-Java-Client/1.0.0");

            if (body != null) {
                conn.setDoOutput(true);
                conn.setRequestProperty("Content-Type", "application/json");

                String jsonBody = gson.toJson(body);
                try (OutputStream os = conn.getOutputStream()) {
                    byte[] input = jsonBody.getBytes(StandardCharsets.UTF_8);
                    os.write(input, 0, input.length);
                }
            }

            int responseCode = conn.getResponseCode();

            // Read response
            StringBuilder response = new StringBuilder();
            try (BufferedReader br = new BufferedReader(
                    new InputStreamReader(
                        responseCode >= 400 ? conn.getErrorStream() : conn.getInputStream(),
                        StandardCharsets.UTF_8))) {
                String responseLine;
                while ((responseLine = br.readLine()) != null) {
                    response.append(responseLine.trim());
                }
            }

            JsonObject jsonResponse = JsonParser.parseString(response.toString()).getAsJsonObject();

            if (responseCode >= 400) {
                String errorMsg = jsonResponse.has("message")
                    ? jsonResponse.get("message").getAsString()
                    : jsonResponse.has("error")
                        ? jsonResponse.get("error").getAsString()
                        : "HTTP " + responseCode;
                throw new IOException("LauraDB API error: " + errorMsg);
            }

            // Check API-level errors
            if (jsonResponse.has("ok") && !jsonResponse.get("ok").getAsBoolean()) {
                String errorMsg = jsonResponse.has("message")
                    ? jsonResponse.get("message").getAsString()
                    : jsonResponse.has("error")
                        ? jsonResponse.get("error").getAsString()
                        : "API request failed";
                throw new IOException("LauraDB API error: " + errorMsg);
            }

            return jsonResponse;
        } finally {
            conn.disconnect();
        }
    }

    /**
     * Get the Gson instance used by this client.
     *
     * @return the Gson instance
     */
    Gson getGson() {
        return gson;
    }

    @Override
    public void close() {
        // No resources to clean up with HttpURLConnection
        // This method is here for future extensibility (e.g., connection pools)
    }
}
