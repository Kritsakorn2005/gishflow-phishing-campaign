package com.gishflow.smsgateway

import android.telephony.SmsManager
import android.util.Base64
import org.json.JSONObject
import java.io.InputStream
import java.io.OutputStream
import java.net.ServerSocket
import java.net.Socket
import java.util.concurrent.Executors

/**
 * Minimal self-contained HTTP server, no third-party libraries. Implements
 * the same contract GishFlow's SendSMSViaGateway already speaks to:
 *   POST /message  (HTTP Basic auth)
 *   {"message": "...", "phoneNumbers": ["+66..."]}
 */
class GatewayServer(
    private val port: Int,
    private val username: String,
    private val password: String,
    private val onLog: (String) -> Unit
) {
    private var serverSocket: ServerSocket? = null
    @Volatile private var running = false
    private val pool = Executors.newCachedThreadPool()

    fun start() {
        running = true
        val socket = ServerSocket(port)
        serverSocket = socket
        pool.execute {
            while (running) {
                try {
                    val client = socket.accept()
                    pool.execute { handle(client) }
                } catch (e: Exception) {
                    if (running) onLog("accept error: ${e.message}")
                }
            }
        }
    }

    fun stop() {
        running = false
        try { serverSocket?.close() } catch (_: Exception) {}
        pool.shutdownNow()
    }

    private fun handle(client: Socket) {
        client.use { sock ->
            try {
                val input = sock.getInputStream()
                val requestLine = readLine(input) ?: return

                var contentLength = 0
                var authHeader: String? = null
                while (true) {
                    val line = readLine(input) ?: break
                    if (line.isEmpty()) break
                    val lower = line.lowercase()
                    if (lower.startsWith("content-length:")) {
                        contentLength = line.substringAfter(":").trim().toIntOrNull() ?: 0
                    } else if (lower.startsWith("authorization:")) {
                        authHeader = line.substringAfter(":").trim()
                    }
                }

                // Read the body as raw bytes (not chars) so multi-byte UTF-8
                // text, such as Thai message bodies, decodes correctly.
                val bodyBytes = ByteArray(contentLength)
                var readTotal = 0
                while (readTotal < contentLength) {
                    val n = input.read(bodyBytes, readTotal, contentLength - readTotal)
                    if (n < 0) break
                    readTotal += n
                }
                val body = String(bodyBytes, 0, readTotal, Charsets.UTF_8)

                val out = sock.getOutputStream()
                val parts = requestLine.split(" ")
                val method = parts.getOrNull(0) ?: ""
                val path = parts.getOrNull(1) ?: ""

                if (method != "POST" || path != "/message") {
                    respond(out, 404, "{\"error\":\"not found\"}")
                    return
                }
                if (!checkAuth(authHeader)) {
                    respond(out, 401, "{\"error\":\"unauthorized\"}")
                    return
                }

                val json = JSONObject(body)
                val message = json.optString("message", "")
                val numbers = json.optJSONArray("phoneNumbers")
                if (message.isEmpty() || numbers == null || numbers.length() == 0) {
                    respond(out, 400, "{\"error\":\"message and phoneNumbers required\"}")
                    return
                }

                val smsManager = SmsManager.getDefault()
                for (i in 0 until numbers.length()) {
                    val phone = numbers.getString(i)
                    try {
                        val chunks = smsManager.divideMessage(message)
                        smsManager.sendMultipartTextMessage(phone, null, chunks, null, null)
                        onLog("Sent to $phone")
                    } catch (e: Exception) {
                        onLog("Failed to send to $phone: ${e.message}")
                    }
                }

                respond(out, 200, "{\"status\":\"ok\"}")
            } catch (e: Exception) {
                onLog("request error: ${e.message}")
            }
        }
    }

    /** Reads a single CRLF (or LF)-terminated line as raw bytes -> ASCII chars. HTTP header lines are always ASCII. */
    private fun readLine(input: InputStream): String? {
        val sb = StringBuilder()
        var any = false
        while (true) {
            val b = input.read()
            if (b == -1) return if (any) sb.toString() else null
            any = true
            if (b == '\n'.code) {
                if (sb.isNotEmpty() && sb.last() == '\r') sb.deleteCharAt(sb.length - 1)
                return sb.toString()
            }
            sb.append(b.toChar())
        }
    }

    private fun checkAuth(header: String?): Boolean {
        if (username.isEmpty() && password.isEmpty()) return true
        if (header == null || !header.startsWith("Basic ")) return false
        val decoded = String(Base64.decode(header.removePrefix("Basic ").trim(), Base64.DEFAULT))
        val idx = decoded.indexOf(':')
        if (idx < 0) return false
        return decoded.substring(0, idx) == username && decoded.substring(idx + 1) == password
    }

    private fun respond(out: OutputStream, code: Int, body: String) {
        val status = when (code) {
            200 -> "200 OK"
            400 -> "400 Bad Request"
            401 -> "401 Unauthorized"
            else -> "404 Not Found"
        }
        val bytes = body.toByteArray(Charsets.UTF_8)
        val header = "HTTP/1.1 $status\r\nContent-Type: application/json\r\nContent-Length: ${bytes.size}\r\nConnection: close\r\n\r\n"
        out.write(header.toByteArray(Charsets.US_ASCII))
        out.write(bytes)
        out.flush()
    }
}
