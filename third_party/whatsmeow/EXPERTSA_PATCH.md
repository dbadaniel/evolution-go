# Expertsa list transport patch

This directory is a source copy of the pinned official `go.mau.fi/whatsmeow` module, version `v0.0.0-20261102152715-54038813c47f` (2026-11-02T15:27:15Z).

Origin recorded by the Go module cache: https://github.com/tulir/whatsmeow.git, commit `54038813c47f3f351eb6fcaafab0e1b99d676850`. Upstream LICENSE, go.mod and go.sum are retained unchanged. This is a local source replacement, not a dependency upgrade.

Only send.go has production changes:

1. Classify an originating ListMessage stanza as `text` before the generic media classification. The encrypted media classification remains `list`.
2. Skip automatic business metadata only when the top-level message is ListMessage and its explicit additional nodes already include a `biz` node containing `list`. Messages without that combination retain the original behavior, including list responses and native-flow messages.

The application supplies a legacy SINGLE_SELECT ListMessage and explicit `biz/list` metadata (`v=2`, `type=product_list`), preserving its reply transport quality-control fields. This combination was confirmed by the user on phone and WhatsApp Web. Tests in expertsa_list_transport_test.go cover the library boundaries; the reviewable patch is ../../patches/whatsmeow-list-transport.patch.

References informing this transport experiment:

- [WPPConnect sendListMessage](https://github.com/wppconnect-team/wa-js/blob/main/src/chat/functions/sendListMessage.ts): originating ListMessage explicitly uses text classification.
- [whatsmeow PR 1146](https://github.com/tulir/whatsmeow/pull/1146): explicit business nodes should override automatically generated business metadata.

These references inform a narrowly scoped local fix; no global business-node override or upstream library refresh is included.
