package com.gishflow.smsgateway

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.net.wifi.WifiManager
import android.os.Bundle
import android.text.format.Formatter
import android.widget.Button
import android.widget.EditText
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat

class MainActivity : AppCompatActivity() {

    private var server: GatewayServer? = null
    private lateinit var statusText: TextView
    private lateinit var logText: TextView
    private lateinit var portInput: EditText
    private lateinit var userInput: EditText
    private lateinit var passInput: EditText

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        statusText = findViewById(R.id.statusText)
        logText = findViewById(R.id.logText)
        portInput = findViewById(R.id.portInput)
        userInput = findViewById(R.id.userInput)
        passInput = findViewById(R.id.passInput)

        findViewById<Button>(R.id.startButton).setOnClickListener { startServer() }
        findViewById<Button>(R.id.stopButton).setOnClickListener { stopServer() }

        if (ContextCompat.checkSelfPermission(this, Manifest.permission.SEND_SMS)
            != PackageManager.PERMISSION_GRANTED
        ) {
            ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.SEND_SMS), 1)
        }

        showIp("Stopped")
    }

    private fun showIp(state: String) {
        val wifiManager = applicationContext.getSystemService(Context.WIFI_SERVICE) as WifiManager
        val ip = Formatter.formatIpAddress(wifiManager.connectionInfo.ipAddress)
        val port = portInput.text.toString()
        statusText.text = "$state - http://$ip:$port"
    }

    private fun appendLog(msg: String) {
        runOnUiThread { logText.append("\n$msg") }
    }

    private fun startServer() {
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.SEND_SMS)
            != PackageManager.PERMISSION_GRANTED
        ) {
            ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.SEND_SMS), 1)
            appendLog("Need SMS permission - grant it, then press Start again")
            return
        }
        val port = portInput.text.toString().toIntOrNull() ?: 8080
        val user = userInput.text.toString()
        val pass = passInput.text.toString()

        server?.stop()
        val s = GatewayServer(port, user, pass) { msg -> appendLog(msg) }
        server = s
        try {
            s.start()
            showIp("Running")
            appendLog("Server started on port $port")
        } catch (e: Exception) {
            appendLog("Failed to start: ${e.message}")
        }
    }

    private fun stopServer() {
        server?.stop()
        server = null
        showIp("Stopped")
        appendLog("Server stopped")
    }

    override fun onDestroy() {
        super.onDestroy()
        server?.stop()
    }
}
