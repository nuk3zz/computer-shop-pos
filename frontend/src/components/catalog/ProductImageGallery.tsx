import { useEffect, useMemo, useRef, useState } from 'react'
import { Boxes, ChevronLeft, ChevronRight, Clock, ImageIcon, Package, Tag, X } from 'lucide-react'
import { getMediaUrl } from '@/lib/media'
import { formatMoney } from '@/lib/shop-settings'
import type { Product } from '@/types'

interface ProductImageGalleryProps {
  product: Product | null
  onClose: () => void
}

export function ProductImageGallery({ product, onClose }: ProductImageGalleryProps) {
  const images = useMemo(() => product ? (product.images?.length ? product.images : product.image_url ? [product.image_url] : []) : [], [product])
  const [activeIndex, setActiveIndex] = useState(0)
  const touchStartX = useRef<number | null>(null)

  const previous = () => setActiveIndex((index) => (index - 1 + images.length) % images.length)
  const next = () => setActiveIndex((index) => (index + 1) % images.length)

  useEffect(() => setActiveIndex(0), [product])

  useEffect(() => {
    if (!product) return
    const previousOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    const handleKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
      if (event.key === 'ArrowLeft' && images.length > 1) previous()
      if (event.key === 'ArrowRight' && images.length > 1) next()
    }
    window.addEventListener('keydown', handleKey)
    return () => {
      document.body.style.overflow = previousOverflow
      window.removeEventListener('keydown', handleKey)
    }
  }, [images.length, onClose, product])

  if (!product) return null

  const durationDays = Math.max(1, Math.ceil(product.preparation_time / 1440))

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/80 p-0 sm:p-5" role="dialog" aria-modal="true" aria-label={`${product.name} details`} onClick={onClose}>
      <div className={`flex w-full flex-col overflow-hidden bg-white text-slate-950 shadow-2xl sm:rounded-xl ${images.length > 0 ? 'h-full max-w-6xl sm:h-[min(820px,92vh)] lg:flex-row' : 'max-h-[92vh] max-w-xl'}`} onClick={(event) => event.stopPropagation()}>
        <section className={`flex flex-col bg-slate-950 text-white ${images.length > 0 ? 'min-h-[42vh] flex-1 lg:min-h-0' : 'h-48 shrink-0'}`}>
          <div className="flex items-center justify-between gap-4 px-4 py-3 lg:px-5">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{product.name}</p>
              <p className="text-xs text-white/60">{images.length > 0 ? `Photo ${activeIndex + 1} of ${images.length}` : 'No photos added'}</p>
            </div>
            <button type="button" onClick={onClose} className="rounded-full bg-white/10 p-2 hover:bg-white/20 lg:hidden" aria-label="Close item details"><X className="h-5 w-5" /></button>
          </div>

          <div
            className="relative flex min-h-0 flex-1 items-center justify-center px-3 pb-3 sm:px-14"
            onTouchStart={(event) => { touchStartX.current = event.changedTouches[0].clientX }}
            onTouchEnd={(event) => {
              if (touchStartX.current === null || images.length < 2) return
              const distance = event.changedTouches[0].clientX - touchStartX.current
              if (Math.abs(distance) > 45) distance > 0 ? previous() : next()
              touchStartX.current = null
            }}
          >
            {images.length > 0 ? (
              <img src={getMediaUrl(images[activeIndex])} alt={`${product.name} photo ${activeIndex + 1}`} className="max-h-full max-w-full select-none object-contain" draggable={false} />
            ) : (
              <div className="flex flex-col items-center gap-3 text-white/35">
                <ImageIcon className="h-16 w-16" />
                <span className="text-sm">No product photos</span>
              </div>
            )}
            {images.length > 1 && (
              <>
                <button type="button" onClick={previous} className="absolute left-3 rounded-full bg-black/50 p-2 hover:bg-white/20 sm:left-5" aria-label="Previous photo"><ChevronLeft className="h-7 w-7" /></button>
                <button type="button" onClick={next} className="absolute right-3 rounded-full bg-black/50 p-2 hover:bg-white/20 sm:right-5" aria-label="Next photo"><ChevronRight className="h-7 w-7" /></button>
              </>
            )}
          </div>

          {images.length > 1 && (
            <div className="flex justify-center gap-2 overflow-x-auto px-4 py-3">
              {images.map((image, index) => (
                <button key={`${image}-${index}`} type="button" onClick={() => setActiveIndex(index)} className={`h-12 w-12 shrink-0 overflow-hidden rounded border-2 ${index === activeIndex ? 'border-white' : 'border-transparent opacity-60'}`} aria-label={`View photo ${index + 1}`}>
                  <img src={getMediaUrl(image)} alt="" className="h-full w-full object-cover" />
                </button>
              ))}
            </div>
          )}
        </section>

        <aside className={`relative overflow-y-auto border-l border-slate-200 bg-white p-5 sm:p-7 ${images.length > 0 ? 'w-full lg:w-[360px] lg:shrink-0' : 'w-full'}`}>
          <button type="button" onClick={onClose} className="absolute right-4 top-4 hidden rounded-full border bg-white p-2 hover:bg-slate-50 lg:block" aria-label="Close item details"><X className="h-5 w-5" /></button>
          <div className="pr-10">
            <div className="mb-2 flex items-center gap-2 text-xs font-medium uppercase tracking-wide text-slate-500">
              {product.item_type === 'service' ? <Clock className="h-4 w-4" /> : <Package className="h-4 w-4" />}
              {product.item_type === 'service' ? 'Service / Repair' : 'Physical Product'}
            </div>
            <h2 className="text-2xl font-bold leading-tight">{product.name}</h2>
            <p className="mt-3 text-xl font-bold text-emerald-700">{formatMoney(product.price)}</p>
          </div>

          <dl className="mt-6 grid grid-cols-2 gap-3 border-y py-4 text-sm">
            <div>
              <dt className="flex items-center gap-1.5 text-xs text-slate-500"><Tag className="h-3.5 w-3.5" /> Category</dt>
              <dd className="mt-1 font-medium">{product.category?.name || 'Uncategorized'}</dd>
            </div>
            <div>
              <dt className="flex items-center gap-1.5 text-xs text-slate-500">{product.item_type === 'service' ? <Clock className="h-3.5 w-3.5" /> : <Boxes className="h-3.5 w-3.5" />} {product.item_type === 'service' ? 'Estimated time' : 'Stock'}</dt>
              <dd className="mt-1 font-medium">{product.item_type === 'service' ? `${durationDays} ${durationDays === 1 ? 'day' : 'days'}` : `${product.stock_quantity} available`}</dd>
            </div>
            {product.sku && (
              <div className="col-span-2">
                <dt className="text-xs text-slate-500">SKU</dt>
                <dd className="mt-1 font-mono text-xs font-medium">{product.sku}</dd>
              </div>
            )}
          </dl>

          <div className="mt-5">
            <h3 className="text-sm font-semibold">Description & compatibility</h3>
            {product.description ? (
              <p className="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-slate-700">{product.description}</p>
            ) : (
              <p className="mt-2 text-sm italic text-slate-400">No description or compatibility information has been added.</p>
            )}
          </div>
        </aside>
      </div>
    </div>
  )
}
