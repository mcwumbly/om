package commands_test

import (
	"errors"

	"github.com/pivotal-cf/jhanda"
	"github.com/pivotal-cf/om/api"
	"github.com/pivotal-cf/om/commands"
	"github.com/pivotal-cf/om/commands/fakes"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeleteProduct", func() {
	var (
		command     commands.DeleteProduct
		fakeService *fakes.DeleteProductService
	)

	BeforeEach(func() {
		fakeService = &fakes.DeleteProductService{}
		command = commands.NewDeleteProduct(fakeService)
	})

	Describe("Execute", func() {
		It("deletes the specific product", func() {
			err := command.Execute([]string{"-p", "some-product-name", "-v", "1.2.3-build.4"})
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeService.DeleteAvailableProductsCallCount()).To(Equal(1))

			input := fakeService.DeleteAvailableProductsArgsForCall(0)
			Expect(input).To(Equal(api.DeleteAvailableProductsInput{
				ProductName:             "some-product-name",
				ProductVersion:          "1.2.3-build.4",
				ShouldDeleteAllProducts: false,
			}))
		})

		Context("failure cases", func() {
			Context("when deleting a product fails", func() {
				It("returns an error", func() {
					fakeService.DeleteAvailableProductsReturns(errors.New("something bad happened"))

					err := command.Execute([]string{"-p", "nah", "-v", "nope"})
					Expect(err).To(MatchError("something bad happened"))
				})
			})

			Context("when the --product-name flag is missing", func() {
				It("returns an error", func() {
					err := command.Execute([]string{
						"--product-version", "1.2.3",
					})
					Expect(err).To(MatchError("could not parse delete-product flags: the required flag `-p, --product-name' was not specified"))
				})
			})

			Context("when the --product-version flag is missing", func() {
				It("returns an error", func() {
					err := command.Execute([]string{
						"--product-name", "some-product",
					})
					Expect(err).To(MatchError("could not parse delete-product flags: the required flag `-v, --product-version' was not specified"))
				})
			})
		})
	})

	Describe("Usage", func() {
		It("returns the usage", func() {
			usage := command.Usage()
			Expect(usage).To(Equal(jhanda.Usage{
				Description:      "This command deletes the named product from the targeted Ops Manager",
				ShortDescription: "deletes a product from the Ops Manager",
				Flags:            command.Options,
			}))
		})
	})
})
