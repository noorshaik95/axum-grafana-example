import { render, screen } from '@testing-library/react'
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from '@/components/ui/card'

describe('Card Components', () => {
  describe('Card', () => {
    it('should render card', () => {
      render(<Card data-testid="card">Card Content</Card>)

      expect(screen.getByTestId('card')).toBeInTheDocument()
    })

    it('should apply custom className', () => {
      render(
        <Card data-testid="card" className="custom-class">
          Card
        </Card>
      )

      expect(screen.getByTestId('card')).toHaveClass('custom-class')
    })
  })

  describe('CardHeader', () => {
    it('should render card header', () => {
      render(<CardHeader data-testid="header">Header</CardHeader>)

      expect(screen.getByTestId('header')).toBeInTheDocument()
    })
  })

  describe('CardTitle', () => {
    it('should render card title', () => {
      render(<CardTitle>Card Title</CardTitle>)

      expect(screen.getByText('Card Title')).toBeInTheDocument()
    })
  })

  describe('CardDescription', () => {
    it('should render card description', () => {
      render(<CardDescription>Card Description</CardDescription>)

      expect(screen.getByText('Card Description')).toBeInTheDocument()
    })
  })

  describe('CardContent', () => {
    it('should render card content', () => {
      render(<CardContent data-testid="content">Content</CardContent>)

      expect(screen.getByTestId('content')).toBeInTheDocument()
    })
  })

  describe('CardFooter', () => {
    it('should render card footer', () => {
      render(<CardFooter data-testid="footer">Footer</CardFooter>)

      expect(screen.getByTestId('footer')).toBeInTheDocument()
    })
  })

  describe('Complete Card', () => {
    it('should render complete card with all components', () => {
      render(
        <Card>
          <CardHeader>
            <CardTitle>Title</CardTitle>
            <CardDescription>Description</CardDescription>
          </CardHeader>
          <CardContent>Content</CardContent>
          <CardFooter>Footer</CardFooter>
        </Card>
      )

      expect(screen.getByText('Title')).toBeInTheDocument()
      expect(screen.getByText('Description')).toBeInTheDocument()
      expect(screen.getByText('Content')).toBeInTheDocument()
      expect(screen.getByText('Footer')).toBeInTheDocument()
    })
  })
})
