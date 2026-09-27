// API Response Types
export interface APIResponse<T = any> {
  success: boolean;
  message: string;
  data?: T;
  error?: string;
}

export interface PaginatedResponse<T = any> {
  success: boolean;
  message: string;
  data: T;
  meta: MetaData;
}

export interface MetaData {
  current_page: number;
  per_page: number;
  total: number;
  total_pages: number;
}

// User Types
export interface User {
  id: string;
  username: string;
  email: string;
  first_name: string;
  last_name: string;
  role: 'admin' | 'manager' | 'sales' | 'technician';
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
}

// Category Types
export interface Category {
  id: string;
  name: string;
  description?: string;
  image_url?: string;
  color?: string;
  sort_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ShopProfile {
  id: number;
  company_name: string;
  logo_url?: string;
  setup_completed: boolean;
  network_mode: 'local' | 'lan';
  auto_backup: boolean;
  backup_time: string;
  created_at: string;
  updated_at: string;
}

export interface InitialSetupInput {
  company_name: string;
  logo_url?: string;
  first_name: string;
  last_name?: string;
  username: string;
  email?: string;
  password: string;
  network_mode: 'local' | 'lan';
}

export interface SystemInfo {
  version: string;
  os: string;
  arch: string;
  native: boolean;
  data_dir: string;
  backup_dir: string;
  network_mode: 'local' | 'lan';
  addresses: string[];
  auto_backup: boolean;
  backup_time: string;
}

export interface BackupInfo {
  name: string;
  kind: 'automatic' | 'manual' | 'imported';
  size: number;
  created_at: string;
}

export interface UpdateInfo {
  current_version: string;
  latest_version?: string;
  update_available: boolean;
  release_name?: string;
  release_url?: string;
}

// Product Types
export interface Product {
  id: string;
  category_id?: string;
  name: string;
  description?: string;
  price: number;
  cost_price: number;
  item_type: 'product' | 'service';
  image_url?: string;
  images: string[];
  barcode?: string;
  sku?: string;
  is_available: boolean;
  stock_quantity: number;
  preparation_time: number;
  sort_order: number;
  created_at: string;
  updated_at: string;
  category?: Category;
}

export interface Customer {
  id: string;
  name: string;
  phone: string;
  image_url?: string;
  email?: string;
  notes?: string;
  order_count: number;
  total_spent: number;
  last_visit?: string;
  created_at: string;
  updated_at: string;
}

export interface CustomerInput {
  name: string;
  phone: string;
  image_url?: string;
  email?: string;
  notes?: string;
}

// Order Types
export interface Order {
  id: string;
  order_number: string;
  user_id?: string;
  customer_id?: string;
  customer_name?: string;
  customer_phone?: string;
  order_type: 'sale' | 'service';
  status: 'pending' | 'confirmed' | 'preparing' | 'ready' | 'served' | 'completed' | 'cancelled';
  subtotal: number;
  tax_amount: number;
  discount_amount: number;
  total_amount: number;
  notes?: string;
  created_at: string;
  updated_at: string;
  served_at?: string;
  completed_at?: string;
  user?: User;
  items?: OrderItem[];
  payments?: Payment[];
}

export interface OrderItem {
  id: string;
  order_id: string;
  product_id: string;
  quantity: number;
  unit_price: number;
  unit_cost: number;
  total_price: number;
  special_instructions?: string;
  status: 'pending' | 'preparing' | 'ready' | 'served';
  created_at: string;
  updated_at: string;
  product?: Product;
  notes?: string; // Alternative field name for special instructions
}

export interface CreateOrderRequest {
  customer_id?: string;
  customer_name?: string;
  customer_phone?: string;
  order_type: 'sale' | 'service';
  items: CreateOrderItem[];
  notes?: string;
}

export interface CreateOrderItem {
  product_id: string;
  quantity: number;
  special_instructions?: string;
}

export interface UpdateOrderStatusRequest {
  status: 'pending' | 'confirmed' | 'preparing' | 'ready' | 'served' | 'completed' | 'cancelled';
  notes?: string;
}

// Payment Types
export interface Payment {
  id: string;
  order_id: string;
  payment_method: 'cash' | 'credit_card' | 'debit_card' | 'digital_wallet';
  amount: number;
  reference_number?: string;
  status: 'pending' | 'completed' | 'failed' | 'refunded';
  processed_by?: string;
  processed_at?: string;
  created_at: string;
  processed_by_user?: User;
}

export interface ProcessPaymentRequest {
  payment_method: 'cash' | 'credit_card' | 'debit_card' | 'digital_wallet';
  amount: number;
  reference_number?: string;
}

export interface PaymentSummary {
  order_id: string;
  total_amount: number;
  total_paid: number;
  pending_amount: number;
  remaining_amount: number;
  is_fully_paid: boolean;
  payment_count: number;
}

// Cart Types (Frontend Only)
export interface CartItem {
  product: Product;
  quantity: number;
  special_instructions?: string;
}

export interface Cart {
  items: CartItem[];
  subtotal: number;
  tax_amount: number;
  total_amount: number;
}

// Dashboard Types
export interface DashboardStats {
  today_orders: number;
  today_revenue: number;
  active_orders: number;
  open_repairs: number;
}

export interface SalesReportItem {
  date: string;
  order_count: number;
  revenue: number;
  profit: number;
}

export interface OrdersReportItem {
  status: string;
  count: number;
  avg_amount: number;
}

// Filter and Query Types
export interface OrderFilters {
  status?: string;
  order_type?: string;
  page?: number;
  per_page?: number;
  customer_id?: string;
}

export interface ProductFilters {
  category_id?: string;
  available?: boolean;
  search?: string;
  page?: number;
  per_page?: number;
}
