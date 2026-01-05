package com.zivpn.app.ui.user

import android.view.LayoutInflater
import android.view.ViewGroup
import androidx.core.content.ContextCompat
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.zivpn.app.R
import com.zivpn.app.data.model.User
import com.zivpn.app.databinding.ItemUserBinding

class UserAdapter(
    private val onDeleteClick: (User) -> Unit,
    private val onRenewClick: (User) -> Unit
) : ListAdapter<User, UserAdapter.ViewHolder>(DiffCallback()) {

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val binding = ItemUserBinding.inflate(LayoutInflater.from(parent.context), parent, false)
        return ViewHolder(binding)
    }

    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        holder.bind(getItem(position))
    }

    inner class ViewHolder(private val binding: ItemUserBinding) : RecyclerView.ViewHolder(binding.root) {
        fun bind(user: User) {
            binding.tvPassword.text = user.password
            binding.tvExpired.text = "Exp: ${user.expired}"
            binding.tvStatus.text = user.status

            val statusColor = when (user.status.lowercase()) {
                "active" -> R.color.connected
                "expired" -> R.color.disconnected
                else -> R.color.connecting
            }
            binding.tvStatus.setTextColor(ContextCompat.getColor(binding.root.context, statusColor))

            binding.btnDelete.setOnClickListener { onDeleteClick(user) }
            binding.btnRenew.setOnClickListener { onRenewClick(user) }
        }
    }

    class DiffCallback : DiffUtil.ItemCallback<User>() {
        override fun areItemsTheSame(oldItem: User, newItem: User) = oldItem.password == newItem.password
        override fun areContentsTheSame(oldItem: User, newItem: User) = oldItem == newItem
    }
}
