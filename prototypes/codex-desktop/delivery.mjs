// Contract spike only. Must run inside a compatible installed host panel.
export async function deliverApprovedFeedback(host, request) {
  if (!host || typeof host.sendFollowUpMessage !== 'function') {
    throw new Error('This desktop surface does not support sending feedback yet.');
  }
  if (!request?.approved || typeof request.id !== 'string' || !request.id.trim() ||
      typeof request.text !== 'string' || !request.text.trim()) {
    throw new Error('Preview and approve a non-empty request before sending.');
  }
  await host.sendFollowUpMessage({
    prompt: `Approved Knit feedback (${request.id}):\n${request.text}`,
    scrollToBottom: false,
  });
  return { request_id: request.id, status: 'delivered', completion_confirmed: false };
}
