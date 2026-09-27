import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { 
  Plus, 
  Search,
  Package,
  Tag,
  Edit,
  Trash2,
  Table,
  Grid3X3,
} from 'lucide-react'
import apiClient from '@/api/client'
import { toastHelpers } from '@/lib/toast-helpers'
import { ProductForm } from '@/components/forms/ProductForm'
import { CategoryForm } from '@/components/forms/CategoryForm'
import { AdminMenuTable } from '@/components/admin/AdminMenuTable'
import { AdminCategoriesTable } from '@/components/admin/AdminCategoriesTable'
import { PaginationControlsComponent } from '@/components/ui/pagination-controls'
import { usePagination } from '@/hooks/usePagination'
import { ProductListSkeleton, CategoryListSkeleton } from '@/components/ui/skeletons'
import { InlineLoading } from '@/components/ui/loading-spinner'
import type { Product, Category } from '@/types'
import { getMediaUrl } from '@/lib/media'
import { formatMoney } from '@/lib/shop-settings'

type DisplayMode = 'table' | 'cards'
type ActiveTab = 'products' | 'categories'

export function AdminMenuManagement() {
  const [displayMode, setDisplayMode] = useState<DisplayMode>('table')
  const [activeTab, setActiveTab] = useState<ActiveTab>('products')
  const [searchTerm, setSearchTerm] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [categorySearch, setCategorySearch] = useState('')
  const [debouncedCategorySearch, setDebouncedCategorySearch] = useState('')
  const [editingProduct, setEditingProduct] = useState<Product | null>(null)
  const [editingCategory, setEditingCategory] = useState<Category | null>(null)
  const [showCreateProductForm, setShowCreateProductForm] = useState(false)
  const [showCreateCategoryForm, setShowCreateCategoryForm] = useState(false)
  const [isSearching, setIsSearching] = useState(false)

  const queryClient = useQueryClient()

  // Pagination hooks
  const productsPagination = usePagination({ 
    initialPage: 1, 
    initialPageSize: 10,
    total: 0 
  })
  
  const categoriesPagination = usePagination({ 
    initialPage: 1, 
    initialPageSize: 10,
    total: 0 
  })

  // Debounce product search
  useEffect(() => {
    if (searchTerm !== debouncedSearch) {
      setIsSearching(true)
    }
    const timer = setTimeout(() => {
      setDebouncedSearch(searchTerm)
      productsPagination.goToFirstPage()
      setIsSearching(false)
    }, 500)
    return () => clearTimeout(timer)
  }, [searchTerm, debouncedSearch])

  // Debounce category search
  useEffect(() => {
    if (categorySearch !== debouncedCategorySearch) {
      setIsSearching(true)
    }
    const timer = setTimeout(() => {
      setDebouncedCategorySearch(categorySearch)
      categoriesPagination.goToFirstPage()
      setIsSearching(false)
    }, 500)
    return () => clearTimeout(timer)
  }, [categorySearch, debouncedCategorySearch])

  // Fetch products with pagination
  const { data: productsData, isLoading: isLoadingProducts, isFetching: isFetchingProducts } = useQuery({
    queryKey: ['admin-products', productsPagination.page, productsPagination.pageSize, debouncedSearch],
    queryFn: () => apiClient.getAdminProducts({
      page: productsPagination.page,
      per_page: productsPagination.pageSize,
      search: debouncedSearch || undefined
    }).then((res: any) => res.data)
  })

  // Fetch categories with pagination
  const { data: categoriesData, isLoading: isLoadingCategories, isFetching: isFetchingCategories } = useQuery({
    queryKey: ['admin-categories', categoriesPagination.page, categoriesPagination.pageSize, debouncedCategorySearch],
    queryFn: () => apiClient.getAdminCategories({
      page: categoriesPagination.page,
      per_page: categoriesPagination.pageSize,
      search: debouncedCategorySearch || undefined
    }).then((res: any) => res.data)
  })

  // Extract data and pagination info
  const products = Array.isArray(productsData) ? productsData : (productsData as any)?.data || []
  const productsPaginationInfo = (productsData as any)?.pagination || { total: 0 }

  const categories = Array.isArray(categoriesData) ? categoriesData : (categoriesData as any)?.data || []
  const categoriesPaginationInfo = (categoriesData as any)?.pagination || { total: 0 }

  // Delete product mutation
  const deleteProductMutation = useMutation({
    mutationFn: ({ id }: { id: string; name: string }) => apiClient.deleteProduct(id),
    onSuccess: (_, product) => {
      queryClient.invalidateQueries({ queryKey: ['admin-products'] })
      toastHelpers.productDeleted(product.name)
    },
    onError: (error: any) => {
      toastHelpers.apiError('Delete product', error)
    }
  })

  // Delete category mutation
  const deleteCategoryMutation = useMutation({
    mutationFn: ({ id }: { id: string; name: string }) => apiClient.deleteCategory(id),
    onSuccess: (_, category) => {
      queryClient.invalidateQueries({ queryKey: ['admin-categories'] })
      toastHelpers.categoryDeleted(category.name)
    },
    onError: (error: any) => {
      toastHelpers.apiError('Delete category', error)
    }
  })

  const handleFormSuccess = () => {
    setShowCreateProductForm(false)
    setShowCreateCategoryForm(false)
    setEditingProduct(null)
    setEditingCategory(null)
  }

  const handleCancelForm = () => {
    setShowCreateProductForm(false)
    setShowCreateCategoryForm(false)
    setEditingProduct(null)
    setEditingCategory(null)
  }

  const handleDeleteProduct = (product: Product) => {
    if (confirm(`Are you sure you want to delete "${product.name}"?`)) {
      deleteProductMutation.mutate({ id: product.id.toString(), name: product.name })
    }
  }

  const handleDeleteCategory = (category: Category) => {
    if (confirm(`Are you sure you want to delete "${category.name}"?`)) {
      deleteCategoryMutation.mutate({ id: category.id.toString(), name: category.name })
    }
  }

  // Show form if creating or editing
  if (showCreateProductForm || editingProduct) {
    return (
      <div className="p-6">
        <ProductForm
          product={editingProduct || undefined}
          mode={editingProduct ? 'edit' : 'create'}
          onSuccess={handleFormSuccess}
          onCancel={handleCancelForm}
        />
      </div>
    )
  }

  if (showCreateCategoryForm || editingCategory) {
    return (
      <div className="p-6">
        <CategoryForm
          category={editingCategory || undefined}
          mode={editingCategory ? 'edit' : 'create'}
          onSuccess={handleFormSuccess}
          onCancel={handleCancelForm}
        />
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Catalog & Inventory</h2>
          <p className="text-muted-foreground">
            Manage computer products, services, images, prices, and categories
          </p>
        </div>
        <div className="flex items-center space-x-4">
          {/* View Toggle */}
          <div className="flex items-center bg-muted rounded-lg p-1">
            <Button
              variant={displayMode === 'table' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setDisplayMode('table')}
              className="px-3"
            >
              <Table className="h-4 w-4 mr-1" />
              Table
            </Button>
            <Button
              variant={displayMode === 'cards' ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setDisplayMode('cards')}
              className="px-3"
            >
              <Grid3X3 className="h-4 w-4 mr-1" />
              Cards
            </Button>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as ActiveTab)} className="w-full">
        <div className="flex items-center justify-between">
          <TabsList className="grid w-[400px] grid-cols-2">
            <TabsTrigger value="products" className="gap-2">
              <Package className="h-4 w-4" />
              Items & Services ({products.length || 0})
            </TabsTrigger>
            <TabsTrigger value="categories" className="gap-2">
              <Tag className="h-4 w-4" />
              Categories ({categories.length || 0})
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Products Tab */}
        <TabsContent value="products" className="space-y-4">
          {/* Search and Add Product */}
          <Card>
            <CardContent className="p-4">
              <div className="flex items-center justify-between gap-4">
                <div className="relative flex-1">
                  <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search items and services by name, category, or description..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="pl-8"
                  />
                  {isSearching && activeTab === 'products' && (
                    <div className="absolute right-2 top-2.5">
                      <InlineLoading size="sm" />
                    </div>
                  )}
                </div>
                <Button onClick={() => setShowCreateProductForm(true)} className="gap-2">
                  <Plus className="h-4 w-4" />
                  Add Item or Service
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Products List */}
          <div className="space-y-4">
            {displayMode === 'table' ? (
              <AdminMenuTable
                data={products}
                categories={categories}
                onEdit={setEditingProduct}
                onDelete={handleDeleteProduct}
                isLoading={isLoadingProducts}
              />
            ) : isLoadingProducts ? (
              <ProductListSkeleton />
            ) : products.length === 0 ? (
              <Card>
                <CardContent className="pt-6">
                  <div className="text-center py-8">
                    <Package className="mx-auto h-12 w-12 text-gray-400" />
                    <h3 className="mt-2 text-sm font-medium text-gray-900">No catalog items</h3>
                    <p className="mt-1 text-sm text-gray-500">
                      {searchTerm ? 'No items match your search.' : 'Get started by adding your first product or service.'}
                    </p>
                    {!searchTerm && (
                      <div className="mt-6">
                        <Button onClick={() => setShowCreateProductForm(true)} className="gap-2">
                          <Plus className="h-4 w-4" />
                          Add Item or Service
                        </Button>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            ) : (
              <div className="grid gap-3 [grid-template-columns:repeat(auto-fill,minmax(260px,1fr))]">
                {products.map((product: Product) => (
                  <Card key={product.id} className="shadow-none transition-colors hover:border-slate-400">
                    <CardContent className="p-3">
                      <div className="flex items-start gap-3">
                        <div className="flex-shrink-0">
                            {product.image_url ? (
                              <img 
                                src={getMediaUrl(product.image_url)}
                                alt={product.name}
                                className="h-14 w-14 rounded-md object-cover"
                              />
                            ) : (
                              <div className="flex h-14 w-14 items-center justify-center rounded-md bg-slate-100">
                                <Package className="h-6 w-6 text-slate-400" />
                              </div>
                            )}
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-start justify-between gap-2">
                            <div className="min-w-0">
                              <h3 className="truncate text-sm font-semibold text-slate-900">{product.name}</h3>
                              <p className="truncate text-xs text-muted-foreground">{product.description || 'No description'}</p>
                            </div>
                            <div className="flex shrink-0 items-center">
                              <Button size="icon" variant="ghost" onClick={() => setEditingProduct(product)} className="h-7 w-7" aria-label={`Edit ${product.name}`}>
                                <Edit className="h-3.5 w-3.5" />
                              </Button>
                              <Button size="icon" variant="ghost" onClick={() => handleDeleteProduct(product)} className="h-7 w-7 text-red-600 hover:text-red-700" aria-label={`Delete ${product.name}`}>
                                <Trash2 className="h-3.5 w-3.5" />
                              </Button>
                            </div>
                          </div>
                          <div className="mt-2 grid grid-cols-2 gap-x-3 gap-y-1 border-t pt-2 text-xs">
                            <div><span className="text-muted-foreground">Sell</span> <span className="font-semibold">{formatMoney(product.price)}</span></div>
                            <div><span className="text-muted-foreground">Cost</span> <span className="font-medium">{formatMoney(product.cost_price)}</span></div>
                            <div><span className="text-muted-foreground">Profit</span> <span className="font-medium text-emerald-700">{formatMoney(product.price - product.cost_price)}</span></div>
                            <div className="text-right sm:text-left">
                              <span className="text-muted-foreground">{product.item_type === 'product' ? 'Stock' : 'Duration'}</span>{' '}
                              <span className="font-medium">{product.item_type === 'product' ? product.stock_quantity : `${Math.max(1, Math.ceil(product.preparation_time / 1440))}d`}</span>
                            </div>
                          </div>
                          <div className="mt-1.5 truncate text-[11px] text-muted-foreground">{product.category?.name || 'Uncategorized'} · {product.item_type === 'product' ? 'Product' : 'Service'}</div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}

            {/* Products Pagination */}
            {products.length > 0 && (
              <div className="mt-6 space-y-4">
                {isFetchingProducts && !isLoadingProducts && (
                  <div className="flex justify-center">
                    <InlineLoading text="Updating products..." />
                  </div>
                )}
                <PaginationControlsComponent
                  pagination={productsPagination}
                  total={productsPaginationInfo.total || products.length}
                />
              </div>
            )}
          </div>
        </TabsContent>

        {/* Categories Tab */}
        <TabsContent value="categories" className="space-y-6">
          {/* Search and Add Category */}
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between gap-4">
                <div className="relative flex-1">
                  <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search categories by name or description..."
                    value={categorySearch}
                    onChange={(e) => setCategorySearch(e.target.value)}
                    className="pl-8"
                  />
                  {isSearching && activeTab === 'categories' && (
                    <div className="absolute right-2 top-2.5">
                      <InlineLoading size="sm" />
                    </div>
                  )}
                </div>
                <Button onClick={() => setShowCreateCategoryForm(true)} className="gap-2">
                  <Plus className="h-4 w-4" />
                  Add Category
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Categories List */}
          <div className="space-y-4">
            {displayMode === 'table' ? (
              <AdminCategoriesTable
                data={categories}
                onEdit={setEditingCategory}
                onDelete={handleDeleteCategory}
                isLoading={isLoadingCategories}
              />
            ) : isLoadingCategories ? (
              <CategoryListSkeleton />
            ) : categories.length === 0 ? (
              <Card>
                <CardContent className="pt-6">
                  <div className="text-center py-8">
                    <Tag className="mx-auto h-12 w-12 text-gray-400" />
                    <h3 className="mt-2 text-sm font-medium text-gray-900">No categories</h3>
                    <p className="mt-1 text-sm text-gray-500">
                      {categorySearch ? 'No categories match your search.' : 'Get started by adding your first category.'}
                    </p>
                    {!categorySearch && (
                      <div className="mt-6">
                        <Button onClick={() => setShowCreateCategoryForm(true)} className="gap-2">
                          <Plus className="h-4 w-4" />
                          Add Category
                        </Button>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            ) : (
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                {categories.map((category: Category) => (
                  <Card key={category.id} className="hover:shadow-md transition-shadow">
                    <CardContent className="pt-6">
                      <div className="text-center">
                        <div 
                          className="mx-auto h-16 w-16 rounded-lg flex items-center justify-center mb-4"
                          style={{ 
                            backgroundColor: category.color || '#6B7280',
                            color: 'white'
                          }}
                        >
                          <Tag className="h-8 w-8" />
                        </div>
                        <h3 className="font-medium text-gray-900 mb-2">{category.name}</h3>
                        <p className="text-sm text-gray-500 mb-4 line-clamp-2">
                          {category.description || "No description"}
                        </p>
                        <div className="flex justify-center space-x-2">
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => setEditingCategory(category)}
                            className="gap-1"
                          >
                            <Edit className="h-4 w-4" />
                            Edit
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => handleDeleteCategory(category)}
                            className="gap-1 text-red-600 hover:text-red-700 hover:border-red-300"
                          >
                            <Trash2 className="h-4 w-4" />
                            Delete
                          </Button>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            )}

            {/* Categories Pagination */}
            {categories.length > 0 && (
              <div className="mt-6 space-y-4">
                {isFetchingCategories && !isLoadingCategories && (
                  <div className="flex justify-center">
                    <InlineLoading text="Updating categories..." />
                  </div>
                )}
                <PaginationControlsComponent
                  pagination={categoriesPagination}
                  total={categoriesPaginationInfo.total || categories.length}
                />
              </div>
            )}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
