<script lang="ts">
	import { tick } from 'svelte';
	import type { ChatItem } from './types.js';
	import { scColorClass } from './types.js';

	let { items }: { items: ChatItem[] } = $props();

	let listEl = $state<HTMLElement | null>(null);

	$effect(() => {
		items.length;
		tick().then(() => {
			if (listEl) listEl.scrollTop = listEl.scrollHeight;
		});
	});

	function badge(authorType: number, privilegeType: number): string {
		if (authorType === 3) return '主播';
		if (authorType === 2) return '管理';
		if (privilegeType === 1) return '总督';
		if (privilegeType === 2) return '提督';
		if (privilegeType === 3) return '舰长';
		return '';
	}

	function badgeClass(authorType: number, privilegeType: number): string {
		if (authorType === 3) return 'owner';
		if (authorType === 2) return 'admin';
		if (privilegeType === 1) return 'guard1';
		if (privilegeType === 2) return 'guard2';
		if (privilegeType === 3) return 'guard3';
		return '';
	}
</script>

<div class="chat-list" bind:this={listEl}>
	{#each items as item (item.data.id)}
		{#if item.kind === 'text'}
			{@const lbl = badge(item.data.authorType, item.data.privilegeType)}
			<div class="row">
				{#if lbl}
					<span class="badge {badgeClass(item.data.authorType, item.data.privilegeType)}">{lbl}</span>
				{/if}
				{#if item.data.medalName && item.data.medalLevel}
					<span class="medal">{item.data.medalName}&nbsp;{item.data.medalLevel}</span>
				{/if}
				<span class="author">{item.data.authorName}</span><span class="sep">: </span>{#if item.data.contentType === 1}<img class="emoticon" src={item.data.contentTypeParams['url']} alt={item.data.content} />{:else}<span class="msg">{item.data.content}</span>{/if}
			</div>
		{:else if item.kind === 'superchat'}
			<div class="row sc-row {scColorClass(item.data.price)}">
				<span class="sc-price">¥{item.data.price}</span>
				<span class="author">{item.data.authorName}</span><span class="sep">: </span><span class="msg">{item.data.content}</span>
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
		gap: 2px;
		padding: 6px 10px;
		scrollbar-width: none;
		font-family: var(--font-family, system-ui, sans-serif);
		font-size: var(--font-size, 15px);
	}
	.chat-list::-webkit-scrollbar {
		display: none;
	}

	.row {
		padding: var(--row-padding, 2px 0);
		border-radius: var(--row-radius, 0px);
		background: var(--row-bg, transparent);
		line-height: 1.5;
		word-break: break-word;
		text-shadow: var(--text-shadow, none);
	}

	/* All inline children flow together so long messages wrap after the username */
	.row > :global(*) {
		display: inline;
	}

	/* SC rows override padding/radius/shadow independently */
	.sc-row {
		padding: var(--sc-row-padding, var(--row-padding, 2px 0));
		border-radius: var(--sc-row-radius, var(--row-radius, 0px));
		border-left: var(--sc-row-border, none);
		text-shadow: var(--sc-text-shadow, var(--text-shadow, none));
	}

	.author {
		font-weight: 700;
		color: var(--author-color, #fff);
		margin-right: var(--author-margin-right, 0px);
	}

	.sep {
		white-space: pre;
		color: var(--sep-color, rgba(255, 255, 255, 0.7));
	}

	.msg {
		color: var(--msg-color, #fff);
	}

	.emoticon {
		height: 24px;
		width: auto;
		vertical-align: middle;
	}

	.medal {
		font-size: var(--medal-font-size, 0.8em);
		padding: var(--medal-padding, 0);
		border-radius: var(--medal-radius, 0);
		background: var(--medal-bg, transparent);
		color: var(--medal-color, rgba(255, 255, 255, 0.65));
	}

	.badge {
		font-size: var(--badge-font-size, 0.82em);
		font-weight: 700;
		padding: var(--badge-padding, 0);
		border-radius: var(--badge-radius, 0);
		margin-right: var(--badge-margin-right, 0px);
	}
	.badge.owner  { background: var(--badge-bg-owner,  transparent); color: var(--badge-color-owner,  #f48fb1); }
	.badge.admin  { background: var(--badge-bg-admin,  transparent); color: var(--badge-color-admin,  #ffcc80); }
	.badge.guard1 { background: var(--badge-bg-guard1, transparent); color: var(--badge-color-guard1, #ce93d8); }
	.badge.guard2 { background: var(--badge-bg-guard2, transparent); color: var(--badge-color-guard2, #90caf9); }
	.badge.guard3 { background: var(--badge-bg-guard3, transparent); color: var(--badge-color-guard3, #80deea); }

	.sc-price {
		font-weight: 700;
		font-size: 0.85em;
		margin-right: 4px;
	}

	/* SC rows override padding/radius/shadow independently */
	.sc-row {
		padding: var(--sc-row-padding, var(--row-padding, 2px 0));
		border-radius: var(--sc-row-radius, var(--row-radius, 0px));
		border-left: var(--sc-row-border, none);
		text-shadow: var(--sc-text-shadow, var(--text-shadow, none));
		/* Read background/color from the SC tier class via custom properties */
		background: var(--sc-bg, transparent);
		color: var(--sc-color, inherit);
	}

	/* SC background/color come from --sc-bg/--sc-color set by .sc-30/.sc-100 etc. in layout.css */
</style>
