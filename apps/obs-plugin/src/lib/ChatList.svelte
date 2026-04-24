<script lang="ts">
	import { tick } from 'svelte';
	import type { ChatItem } from './types.js';
	import { scColorClass } from './types.js';

	let { items }: { items: ChatItem[] } = $props();

	let listEl = $state<HTMLElement | null>(null);

	// Auto-scroll to bottom whenever items change
	$effect(() => {
		// Touch items to subscribe
		items.length;
		tick().then(() => {
			if (listEl) listEl.scrollTop = listEl.scrollHeight;
		});
	});

	function authorLabel(authorType: number, privilegeType: number): string {
		if (authorType === 3) return '主播';
		if (authorType === 2) return '管理';
		if (privilegeType === 1) return '总督';
		if (privilegeType === 2) return '提督';
		if (privilegeType === 3) return '舰长';
		return '';
	}

	function authorLabelClass(authorType: number, privilegeType: number): string {
		if (authorType === 3) return 'badge-owner';
		if (authorType === 2) return 'badge-admin';
		if (privilegeType === 1) return 'badge-guard1';
		if (privilegeType === 2) return 'badge-guard2';
		if (privilegeType === 3) return 'badge-guard3';
		return '';
	}
</script>

<div class="chat-list" bind:this={listEl}>
	{#each items as item (item.data.id)}
		{#if item.kind === 'text'}
			{@const label = authorLabel(item.data.authorType, item.data.privilegeType)}
			<div class="chat-item">
				{#if label}
					<span class="badge {authorLabelClass(item.data.authorType, item.data.privilegeType)}"
						>{label}</span
					>
				{/if}
				{#if item.data.medalName && item.data.medalLevel}
					<span class="medal">{item.data.medalName} {item.data.medalLevel}</span>
				{/if}
				<span class="author">{item.data.authorName}</span>
				<span class="colon">:</span>
				{#if item.data.contentType === 1}
					<!-- emoticon -->
					<img
						class="emoticon"
						src={item.data.contentTypeParams['url']}
						alt={item.data.content}
					/>
				{:else}
					<span class="content">{item.data.content}</span>
				{/if}
			</div>
		{:else if item.kind === 'superchat'}
			<div class="chat-item sc-item {scColorClass(item.data.price)}">
				<span class="sc-price">¥{item.data.price}</span>
				<span class="author">{item.data.authorName}</span>
				<span class="colon">:</span>
				<span class="content">{item.data.content}</span>
			</div>
		{/if}
	{/each}
</div>

<style>
	.chat-list {
		display: flex;
		flex-direction: column;
		overflow-y: auto;
		overflow-x: hidden;
		flex: 1;
		gap: 4px;
		padding: 6px 8px;
		scrollbar-width: none;
	}
	.chat-list::-webkit-scrollbar {
		display: none;
	}

	.chat-item {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px;
		padding: 4px 8px;
		border-radius: 6px;
		background: rgba(0, 0, 0, 0.55);
		font-size: 14px;
		line-height: 1.4;
		word-break: break-word;
	}

	.author {
		font-weight: 700;
		color: #ffffffd9;
	}

	.colon {
		color: #ffffff99;
		margin-right: 2px;
	}

	.content {
		color: #ffffffee;
	}

	.emoticon {
		height: 24px;
		width: auto;
		vertical-align: middle;
	}

	.medal {
		font-size: 11px;
		padding: 1px 5px;
		border-radius: 4px;
		background: rgba(255, 255, 255, 0.15);
		color: #ffffffcc;
	}

	.badge {
		font-size: 11px;
		font-weight: 700;
		padding: 1px 5px;
		border-radius: 4px;
	}
	.badge-owner {
		background: #e91e63;
		color: #fff;
	}
	.badge-admin {
		background: #ff9800;
		color: #fff;
	}
	.badge-guard1 {
		background: #9c27b0;
		color: #fff;
	}
	.badge-guard2 {
		background: #3f51b5;
		color: #fff;
	}
	.badge-guard3 {
		background: #2196f3;
		color: #fff;
	}

	/* SC inline item colors */
	.sc-item {
		border-left: 3px solid currentColor;
	}
	.sc-price {
		font-weight: 700;
		font-size: 12px;
		opacity: 0.9;
	}

	:global(.sc-blue) {
		background: rgba(13, 71, 161, 0.75);
		color: #90caf9;
	}
	:global(.sc-teal) {
		background: rgba(0, 77, 64, 0.75);
		color: #80cbc4;
	}
	:global(.sc-green) {
		background: rgba(27, 94, 32, 0.75);
		color: #a5d6a7;
	}
	:global(.sc-yellow) {
		background: rgba(130, 77, 0, 0.75);
		color: #ffe082;
	}
	:global(.sc-orange) {
		background: rgba(191, 54, 12, 0.75);
		color: #ffcc80;
	}
	:global(.sc-red) {
		background: rgba(183, 28, 28, 0.75);
		color: #ef9a9a;
	}
</style>
