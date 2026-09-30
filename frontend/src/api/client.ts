import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';
import type {
  APIResponse,
  PaginatedResponse,
  LoginRequest,
  LoginResponse,
  User,
  Product,
  Category,
  Order,
  Payment,
  CreateOrderRequest,
  UpdateOrderStatusRequest,
  ProcessPaymentRequest,
  PaymentSummary,
  DashboardStats,
  SalesReportItem,
  OrdersReportItem,
  OrderFilters,
  ProductFilters,
  Customer,
  CustomerInput,
  ShopProfile,
  InitialSetupInput,
  SystemInfo,
  BackupInfo,
  UpdateInfo,
  UpdateDownloadResult,
  Supplier,
  SupplierInput,
  SupplierPurchase,
  SupplierPurchaseInput,
  SupplierTransaction,
  WarrantyClaim,
  WarrantyClaimInput,
  WarrantyClaimUpdate,
} from '@/types';

class APIClient {
  private client: AxiosInstance;

  constructor() {
    const apiUrl = import.meta.env?.VITE_API_URL || '/api/v1';
    console.log('🔧 API Client baseURL:', apiUrl);
    console.log('🔧 Environment VITE_API_URL:', import.meta.env?.VITE_API_URL);
    
    this.client = axios.create({
      baseURL: apiUrl,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Request interceptor to add auth token
    this.client.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('pos_token');
        if (token) {
          config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
      },
      (error) => {
        return Promise.reject(error);
      }
    );

    // Response interceptor to handle auth errors
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          localStorage.removeItem('pos_token');
          localStorage.removeItem('pos_user');
          // Redirect to login page
          window.location.href = '/login';
        }
        return Promise.reject(error);
      }
    );
  }

  // Helper method to handle API responses
  private async request<T>(config: AxiosRequestConfig): Promise<T> {
    try {
      const response: AxiosResponse<T> = await this.client.request(config);
      return response.data;
    } catch (error) {
      if (axios.isAxiosError(error)) {
        throw new Error(error.response?.data?.message || error.message);
      }
      throw error;
    }
  }

  // Authentication endpoints
  async login(credentials: LoginRequest): Promise<APIResponse<LoginResponse>> {
    return this.request({
      method: 'POST',
      url: '/auth/login',
      data: credentials,
    });
  }

  async logout(): Promise<APIResponse> {
    return this.request({
      method: 'POST',
      url: '/auth/logout',
    });
  }

  async getCurrentUser(): Promise<APIResponse<User>> {
    return this.request({
      method: 'GET',
      url: '/auth/me',
    });
  }

  async updateCurrentUser(profile: Pick<User, 'first_name' | 'last_name' | 'username' | 'email'> & { profile_image_url?: string }): Promise<APIResponse<User>> {
    return this.request({ method: 'PUT', url: '/auth/profile', data: profile });
  }

  // Product endpoints
  async getProducts(filters?: ProductFilters): Promise<PaginatedResponse<Product[]>> {
    return this.request({
      method: 'GET',
      url: '/products',
      params: filters,
    });
  }

  async getProduct(id: string): Promise<APIResponse<Product>> {
    return this.request({
      method: 'GET',
      url: `/products/${id}`,
    });
  }

  async getCategories(activeOnly = true): Promise<APIResponse<Category[]>> {
    return this.request({
      method: 'GET',
      url: '/categories',
      params: { active_only: activeOnly },
    });
  }

  async getProductsByCategory(categoryId: string, availableOnly = true): Promise<APIResponse<Product[]>> {
    return this.request({
      method: 'GET',
      url: `/categories/${categoryId}/products`,
      params: { available_only: availableOnly },
    });
  }

  async getCustomers(params?: { search?: string; page?: number; per_page?: number }): Promise<PaginatedResponse<Customer[]>> {
    return this.request({ method: 'GET', url: '/customers', params });
  }

  async getCustomer(id: string): Promise<APIResponse<Customer>> {
    return this.request({ method: 'GET', url: `/customers/${id}` });
  }

  async createCustomer(customer: CustomerInput): Promise<APIResponse<Customer>> {
    return this.request({ method: 'POST', url: '/admin/customers', data: customer });
  }

  async updateCustomer(id: string, customer: CustomerInput): Promise<APIResponse<Customer>> {
    return this.request({ method: 'PUT', url: `/admin/customers/${id}`, data: customer });
  }

  async getSuppliers(): Promise<APIResponse<Supplier[]>> {
    return this.request({ method: 'GET', url: '/suppliers' });
  }

  async createSupplier(input: SupplierInput): Promise<APIResponse<{ id: string }>> {
    return this.request({ method: 'POST', url: '/admin/suppliers', data: input });
  }

  async updateSupplier(id: string, input: SupplierInput): Promise<APIResponse> {
    return this.request({ method: 'PUT', url: `/admin/suppliers/${id}`, data: input });
  }

  async getSupplierPurchases(): Promise<APIResponse<SupplierPurchase[]>> {
    return this.request({ method: 'GET', url: '/supplier-purchases' });
  }

  async getSupplierTransactions(): Promise<APIResponse<SupplierTransaction[]>> {
    return this.request({ method: 'GET', url: '/supplier-transactions' });
  }

  async createSupplierPurchase(input: SupplierPurchaseInput): Promise<APIResponse> {
    return this.request({ method: 'POST', url: '/admin/supplier-purchases', data: input });
  }

  async createSupplierPayment(id: string, input: { amount: number; notes?: string; attachment_url?: string }): Promise<APIResponse> {
    return this.request({ method: 'POST', url: `/admin/suppliers/${id}/payments`, data: input });
  }

  async getSupplierTransactionReference(type: SupplierTransaction['type'], id: string): Promise<Blob> {
    const response = await this.client.get(`/admin/supplier-transactions/${type}/${id}/reference.pdf`, { responseType: 'blob' });
    return response.data;
  }

  async getWarrantyClaims(): Promise<APIResponse<WarrantyClaim[]>> {
    return this.request({ method: 'GET', url: '/warranty-claims' });
  }

  async createWarrantyClaim(input: WarrantyClaimInput): Promise<APIResponse<{ id: string; claim_number: string }>> {
    return this.request({ method: 'POST', url: '/admin/warranty-claims', data: input });
  }

  async updateWarrantyClaim(id: string, input: WarrantyClaimUpdate): Promise<APIResponse> {
    return this.request({ method: 'PATCH', url: `/admin/warranty-claims/${id}`, data: input });
  }

  // Order endpoints
  async getOrders(filters?: OrderFilters): Promise<PaginatedResponse<Order[]>> {
    return this.request({
      method: 'GET',
      url: '/orders',
      params: filters,
    });
  }

  async createOrder(order: CreateOrderRequest): Promise<APIResponse<Order>> {
    return this.request({
      method: 'POST',
      url: '/orders',
      data: order,
    });
  }

  async getOrder(id: string): Promise<APIResponse<Order>> {
    return this.request({
      method: 'GET',
      url: `/orders/${id}`,
    });
  }

  async updateOrderStatus(id: string, status: UpdateOrderStatusRequest['status'], notes?: string): Promise<APIResponse<Order>> {
    const statusUpdate: UpdateOrderStatusRequest = { status, notes };
    return this.request({
      method: 'PATCH',
      url: `/orders/${id}/status`,
      data: statusUpdate,
    });
  }

  async updateFulfillmentStatus(id: string, status: Order['fulfillment_status']): Promise<APIResponse<Order>> {
    return this.request({ method: 'PATCH', url: `/orders/${id}/fulfillment`, data: { status } });
  }

  // Payment endpoints
  async processPayment(orderId: string, payment: ProcessPaymentRequest): Promise<APIResponse<Payment>> {
    return this.request({
      method: 'POST',
      url: `/admin/orders/${orderId}/payments`,
      data: payment,
    });
  }

  async getPayments(orderId: string): Promise<APIResponse<Payment[]>> {
    return this.request({
      method: 'GET',
      url: `/orders/${orderId}/payments`,
    });
  }

  async getPaymentSummary(orderId: string): Promise<APIResponse<PaymentSummary>> {
    return this.request({
      method: 'GET',
      url: `/orders/${orderId}/payment-summary`,
    });
  }

  // Dashboard endpoints
  async getDashboardStats(): Promise<APIResponse<DashboardStats>> {
    return this.request({
      method: 'GET',
      url: '/admin/dashboard/stats',
    });
  }

  async getSalesReport(period: 'today' | 'week' | 'month' = 'today'): Promise<APIResponse<SalesReportItem[]>> {
    return this.request({
      method: 'GET',
      url: '/admin/reports/sales',
      params: { period },
    });
  }

  async getOrdersReport(): Promise<APIResponse<OrdersReportItem[]>> {
    return this.request({
      method: 'GET',
      url: '/admin/reports/orders',
    });
  }

  async getIncomeReport(period: 'today' | 'week' | 'month' | 'year' = 'today'): Promise<APIResponse<any>> {
    return this.request({
      method: 'GET',
      url: '/admin/reports/income',
      params: { period },
    });
  }

  // User management endpoints (Admin only)
  async getUsers(params?: { page?: number; limit?: number; search?: string }): Promise<APIResponse<User[]>> {
    return this.request({
      method: 'GET',
      url: '/admin/users',
      params,
    });
  }

  async createUser(userData: any): Promise<APIResponse<User>> {
    return this.request({
      method: 'POST',
      url: '/admin/users',
      data: userData,
    });
  }

  async updateUser(id: string, userData: any): Promise<APIResponse<User>> {
    return this.request({
      method: 'PUT',
      url: `/admin/users/${id}`,
      data: userData,
    });
  }

  async deleteUser(id: string): Promise<APIResponse> {
    return this.request({
      method: 'DELETE',
      url: `/admin/users/${id}`,
    });
  }

  // Admin-specific product management
  async createProduct(productData: any): Promise<APIResponse<Product>> {
    return this.request({ method: 'POST', url: '/admin/products', data: productData });
  }

  async updateProduct(id: string, productData: any): Promise<APIResponse<Product>> {
    return this.request({ method: 'PUT', url: `/admin/products/${id}`, data: productData });
  }

  async deleteProduct(id: string): Promise<APIResponse> {
    return this.request({ method: 'DELETE', url: `/admin/products/${id}` });
  }

  async uploadProductImage(file: File): Promise<APIResponse<{ url: string; content_type: string; size: number }>> {
    const formData = new FormData();
    formData.append('image', file);

    return this.request({
      method: 'POST',
      url: '/admin/uploads/images',
      data: formData,
      headers: { 'Content-Type': 'multipart/form-data' },
    });
  }

  async uploadSupplierDocument(file: File): Promise<APIResponse<{ url: string; content_type: string; size: number }>> {
    const formData = new FormData();
    formData.append('document', file);
    return this.request({ method: 'POST', url: '/admin/uploads/supplier-documents', data: formData, headers: { 'Content-Type': 'multipart/form-data' } });
  }

  async getShopProfile(): Promise<APIResponse<ShopProfile>> {
    return this.request({ method: 'GET', url: '/shop-profile' });
  }

  async updateShopProfile(profile: { company_name: string; description: string; logo_url?: string }): Promise<APIResponse<ShopProfile>> {
    return this.request({ method: 'PUT', url: '/admin/shop-profile', data: profile });
  }

  async getSetupStatus(): Promise<APIResponse<ShopProfile>> {
    return this.request({ method: 'GET', url: '/setup/status' });
  }

  async uploadSetupLogo(file: File): Promise<APIResponse<{ url: string; content_type: string; size: number }>> {
    const formData = new FormData();
    formData.append('image', file);
    return this.request({ method: 'POST', url: '/setup/upload', data: formData, headers: { 'Content-Type': 'multipart/form-data' } });
  }

  async completeInitialSetup(input: InitialSetupInput): Promise<APIResponse> {
    return this.request({ method: 'POST', url: this.isAuthenticated() ? '/admin/setup/complete' : '/setup/complete', data: input });
  }

  async getSystemInfo(): Promise<APIResponse<SystemInfo>> {
    return this.request({ method: 'GET', url: '/admin/system/info' });
  }

  async updateSystemPreferences(input: { network_mode: 'local' | 'lan'; auto_backup: boolean; backup_time: string }): Promise<APIResponse> {
    return this.request({ method: 'PUT', url: '/admin/system/preferences', data: input });
  }

  async getBackups(): Promise<APIResponse<BackupInfo[]>> {
    return this.request({ method: 'GET', url: '/admin/system/backups' });
  }

  async createBackup(): Promise<APIResponse<BackupInfo>> {
    return this.request({ method: 'POST', url: '/admin/system/backups' });
  }

  async uploadBackup(file: File): Promise<APIResponse<BackupInfo>> {
    const formData = new FormData();
    formData.append('backup', file);
    return this.request({ method: 'POST', url: '/admin/system/backups/upload', data: formData, headers: { 'Content-Type': 'multipart/form-data' } });
  }

  async restoreBackup(name: string): Promise<APIResponse<{ restart_required: boolean }>> {
    return this.request({ method: 'POST', url: '/admin/system/restore', data: { name } });
  }

  async downloadBackup(name: string): Promise<void> {
    const response = await this.client.get(`/admin/system/backups/${encodeURIComponent(name)}/download`, { responseType: 'blob' });
    const url = URL.createObjectURL(response.data);
    const link = document.createElement('a');
    link.href = url;
    link.download = name;
    link.click();
    URL.revokeObjectURL(url);
  }

  async checkForUpdates(): Promise<APIResponse<UpdateInfo>> {
    return this.request({ method: 'GET', url: '/admin/system/updates' });
  }

  async downloadUpdate(): Promise<APIResponse<UpdateDownloadResult>> {
    return this.request({ method: 'POST', url: '/admin/system/updates/download', timeout: 15 * 60 * 1000 });
  }

  async startFresh(confirmation: string): Promise<APIResponse> {
    return this.request({ method: 'POST', url: '/admin/system/start-fresh', data: { confirmation } });
  }

  // Admin-specific category management  
  async createCategory(categoryData: any): Promise<APIResponse<Category>> {
    return this.request({ method: 'POST', url: '/admin/categories', data: categoryData });
  }

  async updateCategory(id: string, categoryData: any): Promise<APIResponse<Category>> {
    return this.request({ method: 'PUT', url: `/admin/categories/${id}`, data: categoryData });
  }

  async deleteCategory(id: string): Promise<APIResponse> {
    return this.request({ method: 'DELETE', url: `/admin/categories/${id}` });
  }

  // Admin products endpoint with pagination
  async getAdminProducts(params?: { page?: number, per_page?: number, limit?: number, search?: string, category_id?: string }): Promise<APIResponse<Product[]>> {
    // Normalize params (handle both per_page and limit)
    const normalizedParams = {
      page: params?.page,
      per_page: params?.per_page || params?.limit,
      search: params?.search,
      category_id: params?.category_id
    }
    
    return this.request({ 
      method: 'GET', 
      url: '/admin/products',
      params: normalizedParams
    });
  }

  // Admin categories endpoint with pagination
  async getAdminCategories(params?: { page?: number, per_page?: number, limit?: number, search?: string, active_only?: boolean }): Promise<APIResponse<Category[]>> {
    // Normalize params (handle both per_page and limit)
    const normalizedParams = {
      page: params?.page,
      per_page: params?.per_page || params?.limit,
      search: params?.search,
      active_only: params?.active_only
    }
    
    return this.request({ 
      method: 'GET', 
      url: '/admin/categories',
      params: normalizedParams
    });
  }

  // Utility methods
  setAuthToken(token: string): void {
    localStorage.setItem('pos_token', token);
  }

  clearAuth(): void {
    localStorage.removeItem('pos_token');
    localStorage.removeItem('pos_user');
  }

  getAuthToken(): string | null {
    return localStorage.getItem('pos_token');
  }

  isAuthenticated(): boolean {
    return !!this.getAuthToken();
  }
}

// Create and export a singleton instance
export const apiClient = new APIClient();
export default apiClient;
