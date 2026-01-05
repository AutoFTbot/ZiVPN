package com.zivpn.app.ui.user

import android.os.Bundle
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ArrayAdapter
import android.widget.EditText
import androidx.fragment.app.Fragment
import androidx.fragment.app.activityViewModels
import androidx.fragment.app.viewModels
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.LinearLayoutManager
import com.google.android.material.dialog.MaterialAlertDialogBuilder
import com.google.android.material.snackbar.Snackbar
import com.zivpn.app.R
import com.zivpn.app.data.model.Server
import com.zivpn.app.data.model.User
import com.zivpn.app.databinding.FragmentUserListBinding
import com.zivpn.app.ui.server.ServerViewModel
import dagger.hilt.android.AndroidEntryPoint
import kotlinx.coroutines.flow.collectLatest
import kotlinx.coroutines.launch

@AndroidEntryPoint
class UserListFragment : Fragment() {

    private var _binding: FragmentUserListBinding? = null
    private val binding get() = _binding!!

    private val userViewModel: UserViewModel by viewModels()
    private val serverViewModel: ServerViewModel by activityViewModels()
    private lateinit var adapter: UserAdapter

    private var currentServer: Server? = null

    override fun onCreateView(inflater: LayoutInflater, container: ViewGroup?, savedInstanceState: Bundle?): View {
        _binding = FragmentUserListBinding.inflate(inflater, container, false)
        return binding.root
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)

        setupRecyclerView()
        setupListeners()
        observeState()
    }

    private fun setupRecyclerView() {
        adapter = UserAdapter(
            onDeleteClick = { user -> confirmDelete(user) },
            onRenewClick = { user -> showRenewDialog(user) }
        )
        binding.rvUsers.adapter = adapter
        binding.rvUsers.layoutManager = LinearLayoutManager(requireContext())
    }

    private fun setupListeners() {
        binding.btnAddUser.setOnClickListener { showAddUserDialog() }
        binding.btnCleanup.setOnClickListener { confirmCleanup() }
    }

    private fun observeState() {
        viewLifecycleOwner.lifecycleScope.launch {
            serverViewModel.uiState.collectLatest { state ->
                if (state.selectedServer != null && state.selectedServer != currentServer) {
                    currentServer = state.selectedServer
                    binding.tvServerName.text = currentServer?.name
                    binding.tvServerDomain.text = currentServer?.domain
                    userViewModel.loadUsers(currentServer!!)
                }

                if (state.servers.isNotEmpty() && currentServer == null) {
                    currentServer = state.servers.first()
                    serverViewModel.selectServer(currentServer!!)
                }
            }
        }

        viewLifecycleOwner.lifecycleScope.launch {
            userViewModel.uiState.collectLatest { state ->
                adapter.submitList(state.users)
                binding.tvEmpty.visibility = if (state.users.isEmpty()) View.VISIBLE else View.GONE
                binding.progress.visibility = if (state.isLoading) View.VISIBLE else View.GONE

                state.error?.let {
                    Snackbar.make(binding.root, it, Snackbar.LENGTH_LONG).show()
                    userViewModel.clearError()
                }

                state.message?.let {
                    Snackbar.make(binding.root, it, Snackbar.LENGTH_SHORT).show()
                    userViewModel.clearMessage()
                }
            }
        }
    }

    private fun showAddUserDialog() {
        val server = currentServer ?: return
        val dialogView = layoutInflater.inflate(R.layout.dialog_add_user, null)
        val etPassword = dialogView.findViewById<EditText>(R.id.et_password)
        val etDays = dialogView.findViewById<EditText>(R.id.et_days)

        MaterialAlertDialogBuilder(requireContext())
            .setTitle("Tambah User")
            .setView(dialogView)
            .setPositiveButton("Buat") { _, _ ->
                val password = etPassword.text.toString().trim()
                val days = etDays.text.toString().toIntOrNull() ?: 30
                if (password.isNotEmpty()) {
                    userViewModel.createUser(server, password, days)
                }
            }
            .setNegativeButton("Batal", null)
            .show()
    }

    private fun confirmDelete(user: User) {
        val server = currentServer ?: return
        MaterialAlertDialogBuilder(requireContext())
            .setTitle("Hapus User")
            .setMessage("Yakin ingin menghapus ${user.password}?")
            .setPositiveButton("Hapus") { _, _ -> userViewModel.deleteUser(server, user.password) }
            .setNegativeButton("Batal", null)
            .show()
    }

    private fun showRenewDialog(user: User) {
        val server = currentServer ?: return
        val dialogView = layoutInflater.inflate(R.layout.dialog_renew_user, null)
        val etDays = dialogView.findViewById<EditText>(R.id.et_days)

        MaterialAlertDialogBuilder(requireContext())
            .setTitle("Perpanjang ${user.password}")
            .setView(dialogView)
            .setPositiveButton("Perpanjang") { _, _ ->
                val days = etDays.text.toString().toIntOrNull() ?: 30
                userViewModel.renewUser(server, user.password, days)
            }
            .setNegativeButton("Batal", null)
            .show()
    }

    private fun confirmCleanup() {
        val server = currentServer ?: return
        MaterialAlertDialogBuilder(requireContext())
            .setTitle("Cleanup Expired")
            .setMessage("Hapus semua akun yang sudah expired?")
            .setPositiveButton("Ya") { _, _ -> userViewModel.cleanupExpired(server) }
            .setNegativeButton("Batal", null)
            .show()
    }

    override fun onDestroyView() {
        super.onDestroyView()
        _binding = null
    }
}
