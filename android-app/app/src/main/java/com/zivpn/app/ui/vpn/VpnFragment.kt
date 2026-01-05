package com.zivpn.app.ui.vpn

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import androidx.fragment.app.Fragment
import androidx.fragment.app.activityViewModels
import androidx.fragment.app.viewModels
import androidx.lifecycle.lifecycleScope
import com.google.android.material.snackbar.Snackbar
import com.zivpn.app.R
import com.zivpn.app.data.model.Server
import com.zivpn.app.data.model.VpnState
import com.zivpn.app.databinding.FragmentVpnBinding
import com.zivpn.app.ui.server.ServerViewModel
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch

@AndroidEntryPoint
class VpnFragment : Fragment() {

    private var _binding: FragmentVpnBinding? = null
    private val binding get() = _binding!!

    private val vpnViewModel: VpnViewModel by viewModels()
    private val serverViewModel: ServerViewModel by activityViewModels()

    private var servers: List<Server> = emptyList()
    private var selectedServer: Server? = null
    private var pendingPassword: String? = null

    private val vpnPermissionLauncher = registerForActivityResult(
        ActivityResultContracts.StartActivityForResult()
    ) { result ->
        if (result.resultCode == Activity.RESULT_OK) {
            pendingPassword?.let { password ->
                selectedServer?.let { server ->
                    vpnViewModel.connect(server, password)
                }
            }
        }
        pendingPassword = null
    }

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentVpnBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        setupListeners()
        observeState()
    }

    private fun setupListeners() {
        binding.btnConnect.setOnClickListener {
            val state = vpnViewModel.uiState.value.vpnState
            if (state == VpnState.CONNECTED || state == VpnState.CONNECTING) {
                vpnViewModel.disconnect()
            } else {
                connect()
            }
        }
    }

    private fun connect() {
        val position = binding.spinnerServer.selectedItemPosition
        if (position < 0 || position >= servers.size) {
            Snackbar.make(binding.root, "Pilih server terlebih dahulu", Snackbar.LENGTH_SHORT).show()
            return
        }

        val password = binding.etPassword.editText?.text?.toString()?.trim() ?: ""
        if (password.isEmpty()) {
            Snackbar.make(binding.root, "Masukkan password", Snackbar.LENGTH_SHORT).show()
            return
        }

        selectedServer = servers[position]
        pendingPassword = password

        val vpnIntent = vpnViewModel.checkVpnPermission()
        if (vpnIntent != null) {
            vpnPermissionLauncher.launch(vpnIntent)
        } else {
            vpnViewModel.connect(selectedServer!!, password)
        }
    }

    private fun observeState() {
        viewLifecycleOwner.lifecycleScope.launch {
            serverViewModel.uiState.collectLatest { state ->
                servers = state.servers
                updateServerSpinner()
            }
        }

        viewLifecycleOwner.lifecycleScope.launch {
            vpnViewModel.uiState.collectLatest { state ->
                updateVpnUI(state.vpnState)

                state.error?.let {
                    Snackbar.make(binding.root, it, Snackbar.LENGTH_LONG).show()
                    vpnViewModel.clearError()
                }
            }
        }
    }

    private fun updateServerSpinner() {
        val serverNames = servers.map { "${it.name} (${it.domain})" }
        val adapter = ArrayAdapter(requireContext(), android.R.layout.simple_spinner_item, serverNames)
        adapter.setDropDownViewResource(android.R.layout.simple_spinner_dropdown_item)
        binding.spinnerServer.adapter = adapter
    }

    private fun updateVpnUI(state: VpnState) {
        val (statusText, statusColor, buttonText, buttonEnabled) = when (state) {
            VpnState.DISCONNECTED -> Quad("Disconnected", R.color.disconnected, "Connect", true)
            VpnState.CONNECTING -> Quad("Connecting...", R.color.connecting, "Cancel", true)
            VpnState.CONNECTED -> Quad("Connected", R.color.connected, "Disconnect", true)
            VpnState.DISCONNECTING -> Quad("Disconnecting...", R.color.connecting, "...", false)
            VpnState.ERROR -> Quad("Error", R.color.disconnected, "Retry", true)
        }

        binding.tvStatus.text = statusText
        binding.tvStatus.setTextColor(ContextCompat.getColor(requireContext(), statusColor))
        binding.btnConnect.text = buttonText
        binding.btnConnect.isEnabled = buttonEnabled

        val connectedServer = vpnViewModel.uiState.value.connectedServer
        binding.tvServerInfo.text = if (state == VpnState.CONNECTED && connectedServer != null) {
            "${connectedServer.name} - ${connectedServer.domain}"
        } else {
            "Pilih server untuk connect"
        }

        // Disable inputs when connected
        binding.spinnerServer.isEnabled = state == VpnState.DISCONNECTED || state == VpnState.ERROR
        binding.etPassword.isEnabled = state == VpnState.DISCONNECTED || state == VpnState.ERROR
    }

    data class Quad<A, B, C, D>(val first: A, val second: B, val third: C, val fourth: D)

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
