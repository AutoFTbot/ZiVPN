package com.zivpn.app.ui.server

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.fragment.app.Fragment
import androidx.fragment.app.viewModels
import androidx.lifecycle.lifecycleScope
import androidx.navigation.fragment.findNavController
import com.google.android.material.snackbar.Snackbar
import com.zivpn.app.databinding.FragmentAddServerBinding
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch

@AndroidEntryPoint
class AddServerFragment : Fragment() {

    private var _binding: FragmentAddServerBinding? = null
    private val binding get() = _binding!!

    private val viewModel: ServerViewModel by viewModels()

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentAddServerBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        binding.btnSave.setOnClickListener { saveServer() }

        viewLifecycleOwner.lifecycleScope.launch {
            viewModel.uiState.collectLatest { state ->
                binding.progress.visibility = if (state.isLoading) View.VISIBLE else View.GONE
                binding.btnSave.isEnabled = !state.isLoading

                state.error?.let {
                    Snackbar.make(binding.root, it, Snackbar.LENGTH_LONG).show()
                    viewModel.clearError()
                }

                state.message?.let {
                    Snackbar.make(binding.root, it, Snackbar.LENGTH_SHORT).show()
                    viewModel.clearMessage()
                    findNavController().popBackStack()
                }
            }
        }
    }

    private fun saveServer() {
        val name = binding.etName.editText?.text?.toString()?.trim() ?: ""
        val domain = binding.etDomain.editText?.text?.toString()?.trim() ?: ""
        val port = binding.etPort.editText?.text?.toString()?.toIntOrNull() ?: 5667
        val apiPort = binding.etApiPort.editText?.text?.toString()?.toIntOrNull() ?: 8080
        val apiKey = binding.etApiKey.editText?.text?.toString()?.trim() ?: ""
        val obfs = binding.etObfs.editText?.text?.toString()?.trim() ?: "zivpn"

        if (name.isEmpty() || domain.isEmpty() || apiKey.isEmpty()) {
            Snackbar.make(binding.root, "Lengkapi semua field", Snackbar.LENGTH_SHORT).show()
            return
        }

        viewModel.addServer(name, domain, port, apiPort, apiKey, obfs)
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
