import { useEffect, useRef } from 'react'

export function useIntersectionObserver<T extends HTMLElement>(options = {}) {
  const elementRef = useRef<T>(null)

  useEffect(() => {
    const element = elementRef.current
    if (!element) return

    const observer = new IntersectionObserver((entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          element.style.animationPlayState = 'running'
        }
      })
    }, {
      threshold: 0.1,
      ...options
    })

    observer.observe(element)

    return () => {
      observer.unobserve(element)
    }
  }, [options])

  return elementRef
} 