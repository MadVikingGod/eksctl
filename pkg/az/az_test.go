package az_test

import (
	"fmt"

	. "github.com/weaveworks/eksctl/pkg/az"
	"github.com/weaveworks/eksctl/pkg/eks"
	"github.com/weaveworks/eksctl/pkg/testutils"

	"github.com/aws/aws-sdk-go/aws"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/aws/aws-sdk-go/service/ec2"
)

var _ = Describe("AZ", func() {

	Describe("When calling SelectZones", func() {
		var (
			zonesToAvoid []*ec2.AvailabilityZone
			c            *eks.ClusterProvider
			p            *testutils.MockProvider
			err          error
		)

		BeforeEach(func() {
			zonesToAvoid = avoidedZones(ec2.AvailabilityZoneStateAvailable)
		})

		Context("with a region that has no zones to avoid", func() {
			var (
				zones  []*ec2.AvailabilityZone
				region *string
			)

			Context("and all zones available", func() {
				var (
					selectedZones []string
					azSelector    *AvailabilityZoneSelector
				)
				BeforeEach(func() {
					region = aws.String("us-west-2")

					zones = usWest2Zones(ec2.AvailabilityZoneStateAvailable)
					c, p = createProviders()

					p.MockEC2().On("DescribeAvailabilityZones",
						mock.MatchedBy(func(input *ec2.DescribeAvailabilityZonesInput) bool {
							filter := input.Filters[0]
							return *filter.Name == "region-name" && *filter.Values[0] == *region
						}),
					).Return(&ec2.DescribeAvailabilityZonesOutput{
						AvailabilityZones: zones,
					}, nil)

					azSelector = NewSelectorWithDefaults(p.MockEC2())
				})

				JustBeforeEach(func() {
					selectedZones, err = azSelector.SelectZones(*region)
				})

				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})

				It("should have called AWS EC2 DescribeAvailabilityZones", func() {
					Expect(p.MockEC2().AssertNumberOfCalls(GinkgoT(), "DescribeAvailabilityZones", 1)).To(BeTrue())
				})

				It("should have returned 3 availability zones", func() {
					Expect(len(selectedZones)).To(Equal(3))
				})
			})

			Context("and only 1 zone is available", func() {
				var (
					selectedZones    []string
					expectedZoneName *string
					azSelector       *AvailabilityZoneSelector
				)
				BeforeEach(func() {
					westZone := usWest2Zones(ec2.AvailabilityZoneStateAvailable)[0]
					expectedZoneName = westZone.ZoneName
					zones = []*ec2.AvailabilityZone{westZone}

					c, p = createProviders()

					p.MockEC2().On("DescribeAvailabilityZones",
						mock.MatchedBy(func(input *ec2.DescribeAvailabilityZonesInput) bool {
							filter := input.Filters[0]
							return *filter.Name == "region-name" && *filter.Values[0] == *region
						}),
					).Return(&ec2.DescribeAvailabilityZonesOutput{
						AvailabilityZones: zones,
					}, nil)

					azSelector = NewSelectorWithDefaults(p.MockEC2())
				})

				JustBeforeEach(func() {
					selectedZones, err = azSelector.SelectZones(*region)
				})

				It("should not error", func() {
					Expect(err).NotTo(HaveOccurred())
				})

				It("should have called AWS EC2 DescribeAvailabilityZones", func() {
					Expect(p.MockEC2().AssertNumberOfCalls(GinkgoT(), "DescribeAvailabilityZones", 1)).To(BeTrue())
				})

				It("should have returned 3 identical availability zones", func() {
					Expect(len(selectedZones)).To(Equal(3))

					for _, actualZoneName := range selectedZones {
						Expect(actualZoneName).To(Equal(*expectedZoneName))
					}
				})
			})
		})

		Context("with a region that has zones to avoid", func() {
			var (
				zones            []*ec2.AvailabilityZone
				selectedZones    []string
				region           *string
				azSelector       *AvailabilityZoneSelector
				expectedZoneName *string
			)
			BeforeEach(func() {
				region = aws.String("us-east-1")
				expectedZoneName = aws.String("us-east1-c")

				zones = usEast1Zones(ec2.AvailabilityZoneStateAvailable)
				c, p = createProviders()

				p.MockEC2().On("DescribeAvailabilityZones",
					mock.MatchedBy(func(input *ec2.DescribeAvailabilityZonesInput) bool {
						filter := input.Filters[0]
						return *filter.Name == "region-name" && *filter.Values[0] == *region
					}),
				).Return(&ec2.DescribeAvailabilityZonesOutput{
					AvailabilityZones: zones,
				}, nil)

				azSelector = NewSelectorWithDefaults(p.MockEC2())
			})

			JustBeforeEach(func() {
				selectedZones, err = azSelector.SelectZones(*region)
			})

			It("should not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})

			It("should have called AWS EC2 DescribeAvailabilityZones", func() {
				Expect(p.MockEC2().AssertNumberOfCalls(GinkgoT(), "DescribeAvailabilityZones", 1)).To(BeTrue())
			})

			It("should have returned 3 availability zones", func() {
				Expect(len(selectedZones)).To(Equal(3))
			})

			It("should have returned none of the zones to avoid", func() {
				for _, actualZoneName := range selectedZones {
					Expect(actualZoneName).To(Equal(*expectedZoneName))
				}
			})
		})

		Context("with an error from AWS", func() {
			var (
				selectedZones []string
				azSelector    *AvailabilityZoneSelector
			)
			BeforeEach(func() {
				c, p = createProviders()

				p.MockEC2().On("DescribeAvailabilityZones",
					mock.MatchedBy(func(input *ec2.DescribeAvailabilityZonesInput) bool {
						// This will match an valid DescribeAvailabilityZonesInput
						return true
					}),
				).Return(nil, fmt.Errorf("Some random error from AWS"))

				azSelector = NewSelectorWithDefaults(p.MockEC2())
			})

			JustBeforeEach(func() {
				selectedZones, err = azSelector.SelectZones("us-west-2")
			})

			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
			})

			It("should not have returned selected zones", func() {
				Expect(selectedZones).Should(BeNil())
			})

			It("should have called AWS EC2 DescribeAvailabilityZones", func() {
				Expect(p.MockEC2().AssertNumberOfCalls(GinkgoT(), "DescribeAvailabilityZones", 1)).To(BeTrue())
			})
		})
	})
})

func createProviders() (*eks.ClusterProvider, *testutils.MockProvider) {
	p := testutils.NewMockProvider()

	c := &eks.ClusterProvider{
		Provider: p,
		Spec: &eks.ClusterConfig{
			Region: "us-west-1",
		},
	}

	return c, p
}

func createAvailabilityZone(region string, state string, zone string) *ec2.AvailabilityZone {
	return &ec2.AvailabilityZone{
		RegionName: aws.String(region),
		State:      aws.String(state),
		ZoneName:   aws.String(zone),
	}
}

func avoidedZones(initialStatus string) []*ec2.AvailabilityZone {
	return []*ec2.AvailabilityZone{
		createAvailabilityZone("US East (N. Virginia)", initialStatus, "us-east1-a"),
		createAvailabilityZone("US East (N. Virginia)", initialStatus, "us-east1-b"),
	}
}

func usEast1Zones(initialStatus string) []*ec2.AvailabilityZone {
	return append(avoidedZones(initialStatus), createAvailabilityZone("US East (N. Virginia)", initialStatus, "us-east1-c"))
}

func usWest2Zones(initialStatus string) []*ec2.AvailabilityZone {
	return []*ec2.AvailabilityZone{
		createAvailabilityZone("US West (N. California)", initialStatus, "us-west2-a"),
		createAvailabilityZone("US West (N. California)", initialStatus, "us-west2-b"),
		createAvailabilityZone("US West (N. California)", initialStatus, "us-west2-c"),
	}
}