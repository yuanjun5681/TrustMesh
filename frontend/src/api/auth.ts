import { api } from './client'
import type {
  ApiResponse,
  AuthRegisterRequest,
  AuthLoginRequest,
  AuthSuccessData,
} from '@/types'

export async function register(input: AuthRegisterRequest) {
  return api.post('auth/register', { json: input }).json<ApiResponse<AuthSuccessData>>()
}

export async function login(input: AuthLoginRequest) {
  return api.post('auth/login', { json: input }).json<ApiResponse<AuthSuccessData>>()
}
