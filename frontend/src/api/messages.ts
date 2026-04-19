// messages.ts — API client functions for the messaging system
import { api } from './client';
import type { MessageListResponse, ConversationListResponse } from '../types/api';

export function sendMessage(recvrSeq: number, content: string) {
  return api.post('/api/messages', { recvrSeq, content });
}

export function getInbox(page: number, size: number) {
  return api.get<MessageListResponse>(`/api/messages/inbox?page=${page}&size=${size}`);
}

export function getOutbox(page: number, size: number) {
  return api.get<MessageListResponse>(`/api/messages/outbox?page=${page}&size=${size}`);
}

export function markAsRead(amSeq: number) {
  return api.put(`/api/messages/${amSeq}/read`);
}

export function deleteMessage(amSeq: number) {
  return api.del(`/api/messages/${amSeq}`);
}

export function getConversations() {
  return api.get<ConversationListResponse>('/api/messages/conversations');
}

export function getConversationMessages(otherSeq: number, page: number, size: number) {
  return api.get<MessageListResponse>(`/api/messages/conversations/${otherSeq}?page=${page}&size=${size}`);
}

export function markConversationRead(otherSeq: number) {
  return api.put(`/api/messages/conversations/${otherSeq}/read`);
}
