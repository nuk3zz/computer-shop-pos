import { useEffect, useState } from 'react'
import type { ChangeEvent } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Form } from '@/components/ui/form'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  TextInputField,
  TextareaField,
  PriceInputField,
  NumberInputField,
  SelectField,
  FormSubmitButton,
  productStatusOptions
} from '@/components/forms/FormComponents'
import { createProductSchema, updateProductSchema, type CreateProductData, type UpdateProductData } from '@/lib/form-schemas'
import { toastHelpers } from '@/lib/toast-helpers'
import apiClient from '@/api/client'
import { getMediaUrl } from '@/lib/media'
import type { Product } from '@/types'
import { ImagePlus, Trash2, X } from 'lucide-react'

interface ProductFormProps {
  product?: Product // If provided, we're editing; otherwise creating
  onSuccess?: () => void
  onCancel?: () => void
  mode?: 'create' | 'edit'
}

export function ProductForm({ product, onSuccess, onCancel, mode = 'create' }: ProductFormProps) {
  const queryClient = useQueryClient()
  const isEditing = mode === 'edit' && product
  const [existingImages, setExistingImages] = useState<string[]>(
    product?.images?.length ? product.images : product?.image_url ? [product.image_url] : [],
  )
  const [newImages, setNewImages] = useState<Array<{ file: File; preview: string }>>([])
  const [imageError, setImageError] = useState('')

  // Fetch categories for dropdown
  const { data: categories = [] } = useQuery({
    queryKey: ['categories'],
    queryFn: () => apiClient.getCategories().then((res) => res.data)
  })

  // Create category options for select field
  const categoryOptions = categories.map((cat) => ({
    value: cat.id.toString(),
    label: cat.name
  }))

  // Choose the appropriate schema and default values
  const schema = isEditing ? updateProductSchema : createProductSchema
  const defaultValues = isEditing
    ? {
        id: product.id,
        name: product.name,
        description: product.description || '',
        price: product.price,
        cost_price: product.cost_price || 0,
        item_type: product.item_type || (product.preparation_time > 0 ? ('service' as const) : ('product' as const)),
        category_id: product.category_id,
        image_url: product.image_url || '',
        image_urls: product.images || (product.image_url ? [product.image_url] : []),
        status: product.is_available ? ('active' as const) : ('inactive' as const),
        stock_quantity: product.stock_quantity || 0,
        preparation_time: product.item_type === 'service' ? Math.max(1, Math.ceil(product.preparation_time / 1440)) : 0
      }
    : {
        name: '',
        description: '',
        price: 0,
        cost_price: 0,
        item_type: 'product' as const,
        category_id: categories[0]?.id || '',
        image_url: '',
        image_urls: [],
        status: 'active' as const,
        stock_quantity: 0,
        preparation_time: 0
      }

  const form = useForm<CreateProductData | UpdateProductData>({
    resolver: zodResolver(schema),
    defaultValues
  })

  useEffect(() => {
    if (!isEditing && categories.length > 0 && !form.getValues('category_id')) {
      form.setValue('category_id', categories[0].id)
    }
  }, [categories, form, isEditing])

  const itemType = form.watch('item_type')

  useEffect(() => {
    const duration = form.getValues('preparation_time') || 0
    if (itemType === 'service' && duration < 1) {
      form.setValue('preparation_time', 1)
    }
    if (itemType === 'product' && duration !== 0) {
      form.setValue('preparation_time', 0)
    }
  }, [form, itemType])

  const handleImageChange = (event: ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files || [])
    if (files.length === 0) return

    const acceptedTypes = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
    if (files.some((file) => !acceptedTypes.includes(file.type))) {
      setImageError('Use a JPG, PNG, WebP, or GIF image.')
      event.target.value = ''
      return
    }
    if (files.some((file) => file.size > 5 * 1024 * 1024)) {
      setImageError('Each image must be 5 MB or smaller.')
      event.target.value = ''
      return
    }
    if (existingImages.length + newImages.length + files.length > 10) {
      setImageError('A listing can have up to 10 photos.')
      event.target.value = ''
      return
    }

    setImageError('')
    setNewImages((current) => [
      ...current,
      ...files.map((file) => ({ file, preview: URL.createObjectURL(file) })),
    ])
    event.target.value = ''
  }

  const prepareProductData = async (data: Partial<CreateProductData>) => {
    const { status, ...productData } = data
    const preparedData = {
      ...productData,
      stock_quantity: productData.item_type === 'product' ? productData.stock_quantity || 0 : 0,
      preparation_time: productData.item_type === 'service'
        ? Math.max(1, productData.preparation_time || 1) * 1440
        : 0,
      is_available: status ? status === 'active' : undefined
    }

    const uploadedImages = await Promise.all(newImages.map(async ({ file }) => {
      const uploadResponse = await apiClient.uploadProductImage(file)
      return uploadResponse.data?.url || ''
    }))
    const imageURLs = [...existingImages, ...uploadedImages.filter(Boolean)].slice(0, 10)
    preparedData.image_urls = imageURLs
    preparedData.image_url = imageURLs[0] || ''

    return preparedData
  }

  // Create mutation
  const createMutation = useMutation({
    mutationFn: async (data: CreateProductData) => apiClient.createProduct(await prepareProductData(data)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-products'] })
      queryClient.invalidateQueries({ queryKey: ['products'] })
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      toastHelpers.productCreated(form.getValues('name') || 'Product')
      form.reset()
      onSuccess?.()
    },
    onError: (error) => {
      toastHelpers.apiError('Create product', error)
    }
  })

  // Update mutation
  const updateMutation = useMutation({
    mutationFn: async (data: UpdateProductData) => {
      const { id, ...editableData } = data
      return apiClient.updateProduct(id.toString(), await prepareProductData(editableData))
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-products'] })
      queryClient.invalidateQueries({ queryKey: ['products'] })
      queryClient.invalidateQueries({ queryKey: ['categories'] })
      toastHelpers.apiSuccess('Update', `Product "${form.getValues('name')}"`)
      onSuccess?.()
    },
    onError: (error) => {
      toastHelpers.apiError('Update product', error)
    }
  })

  const onSubmit = (data: CreateProductData | UpdateProductData) => {
    if (isEditing) {
      updateMutation.mutate(data as UpdateProductData)
    } else {
      createMutation.mutate(data as CreateProductData)
    }
  }

  const isLoading = createMutation.isPending || updateMutation.isPending

  if (categories.length === 0) {
    return (
      <Card className="w-full max-w-2xl mx-auto">
        <CardContent className="pt-6">
          <div className="text-center py-8">
            <p className="text-muted-foreground mb-4">You need to create at least one category before adding products.</p>
            <Button onClick={onCancel} variant="outline">
              Go Back
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="w-full max-w-2xl mx-auto">
      <CardHeader className="flex flex-row items-center justify-between">
        <CardTitle>{isEditing ? 'Edit Catalog Item' : 'Create Catalog Item'}</CardTitle>
        {onCancel && (
          <Button variant="ghost" size="icon" onClick={onCancel} disabled={isLoading}>
            <X className="h-4 w-4" />
          </Button>
        )}
      </CardHeader>
      <CardContent>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
            {/* Basic Information */}
            <div className="space-y-4">
              <TextInputField
                control={form.control}
                name="name"
                label="Item or Service Name"
                placeholder="Example: 16 GB DDR4 RAM or Windows Installation"
                description="The name shown in your sales and service catalog"
              />

              <TextareaField
                control={form.control}
                name="description"
                label="Description"
                placeholder="Describe the product..."
                rows={3}
                description="Optional description for staff and customers"
              />

              <div className="space-y-2">
                <div className="flex items-center justify-between gap-3">
                  <label className="text-sm font-medium">Listing photos</label>
                  <span className="text-xs text-muted-foreground">{existingImages.length + newImages.length}/10</span>
                </div>
                {(existingImages.length > 0 || newImages.length > 0) && (
                  <div className="grid grid-cols-3 gap-2 sm:grid-cols-5">
                    {existingImages.map((imageURL, index) => (
                      <div key={`${imageURL}-${index}`} className="relative aspect-square overflow-hidden rounded-md border bg-muted">
                        <img src={getMediaUrl(imageURL)} alt={`Listing photo ${index + 1}`} className="h-full w-full object-cover" />
                        {index === 0 && <span className="absolute bottom-1 left-1 rounded bg-black/70 px-1.5 py-0.5 text-[10px] text-white">Cover</span>}
                        <button type="button" aria-label={`Remove photo ${index + 1}`} onClick={() => setExistingImages((images) => images.filter((_, imageIndex) => imageIndex !== index))} className="absolute right-1 top-1 rounded bg-white/90 p-1 text-slate-700 shadow hover:text-destructive">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    ))}
                    {newImages.map(({ preview }, index) => (
                      <div key={preview} className="relative aspect-square overflow-hidden rounded-md border bg-muted">
                        <img src={preview} alt={`New listing photo ${index + 1}`} className="h-full w-full object-cover" />
                        {existingImages.length === 0 && index === 0 && <span className="absolute bottom-1 left-1 rounded bg-black/70 px-1.5 py-0.5 text-[10px] text-white">Cover</span>}
                        <button type="button" aria-label={`Remove new photo ${index + 1}`} onClick={() => setNewImages((images) => {
                          URL.revokeObjectURL(images[index].preview)
                          return images.filter((_, imageIndex) => imageIndex !== index)
                        })} className="absolute right-1 top-1 rounded bg-white/90 p-1 text-slate-700 shadow hover:text-destructive">
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
                <div className="flex items-center gap-3">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-md border bg-muted"><ImagePlus className="h-5 w-5 text-muted-foreground" /></div>
                  <Input type="file" multiple accept="image/jpeg,image/png,image/webp,image/gif" onChange={handleImageChange} disabled={isLoading || existingImages.length + newImages.length >= 10} />
                </div>
                <p className="text-xs text-muted-foreground">Choose up to 10 photos. The first is the catalog cover; maximum 5 MB each.</p>
                {imageError && <p className="text-sm text-destructive">{imageError}</p>}
              </div>
            </div>

            <SelectField
              control={form.control}
              name="item_type"
              label="Catalog Type"
              options={[
                { value: 'product', label: 'Physical Product' },
                { value: 'service', label: 'Service / Repair' },
              ]}
              description="Services automatically create trackable repair tickets"
            />

            {/* Pricing & Details */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <PriceInputField control={form.control} name="cost_price" label="Actual Cost (LKR)" currency="Rs." description="What you paid; use 0 for time-based services" />

              <PriceInputField control={form.control} name="price" label="Selling Price (LKR)" currency="Rs." description="Amount charged to the customer" />

              {itemType === 'product' ? (
                <NumberInputField
                  control={form.control}
                  name="stock_quantity"
                  label="Quantity in Stock"
                  min={0}
                  max={1000000}
                  description="How many of this physical item you currently own"
                />
              ) : (
                <NumberInputField
                  control={form.control}
                  name="preparation_time"
                  label="Estimated Service Duration (days)"
                  min={1}
                  max={365}
                  description="Minimum 1 day"
                />
              )}
            </div>

            {/* Category & Status */}
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <SelectField
                control={form.control}
                name="category_id"
                label="Category"
                options={categoryOptions}
                placeholder="Select a category"
                description="Category used to find the item quickly"
              />

              <SelectField
                control={form.control}
                name="status"
                label="Status"
                options={productStatusOptions}
                description="Available items appear in Sales & Services"
              />
            </div>

            {/* Action Buttons */}
            <div className="flex gap-3 pt-4">
              <FormSubmitButton isLoading={isLoading} loadingText={isEditing ? 'Updating...' : 'Creating...'} className="flex-1">
                {isEditing ? 'Update Item' : 'Create Item'}
              </FormSubmitButton>

              {onCancel && (
                <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading} className="flex-1">
                  Cancel
                </Button>
              )}
            </div>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}
