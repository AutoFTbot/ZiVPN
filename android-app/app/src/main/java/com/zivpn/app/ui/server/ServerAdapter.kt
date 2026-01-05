package com.zivpn.app.ui.server

import android.view.LayoutInflater
import android.view.ViewGroup
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import com.zivpn.app.data.model.Server
import com.zivpn.app.databinding.ItemServerBinding

class ServerAdapter(
    private val onItemClick: (Server) -> Unit,
    private val onDeleteClick: (Server) -> Unit
) : ListAdapter<Server, ServerAdapter.ViewHolder>(DiffCallback()) {

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): ViewHolder {
        val binding = ItemServerBinding.inflate(LayoutInflater.from(parent.context), parent, false)
        return ViewHolder(binding)
    }

    override fun onBindViewHolder(holder: ViewHolder, position: Int) {
        holder.bind(getItem(position))
    }

    inner class ViewHolder(private val binding: ItemServerBinding) : RecyclerView.ViewHolder(binding.root) {
        fun bind(server: Server) {
            binding.tvName.text = server.name
            binding.tvDomain.text = "${server.domain}:${server.port}"
            binding.root.setOnClickListener { onItemClick(server) }
            binding.btnDelete.setOnClickListener { onDeleteClick(server) }
        }
    }

    class DiffCallback : DiffUtil.ItemCallback<Server>() {
        override fun areItemsTheSame(oldItem: Server, newItem: Server) = oldItem.id == newItem.id
        override fun areContentsTheSame(oldItem: Server, newItem: Server) = oldItem == newItem
    }
}
