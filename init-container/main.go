package main

import (
    "fmt"
    "os"
    "strings"

    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/arn"
    "github.com/aws/aws-sdk-go/aws/awserr"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/secretsmanager"
    "k8s.io/klog"
)

func main() {
    envSecretArns := os.Getenv("SECRET_ARNS")
    var AWSRegion string
    klog.Info("SECRET_ARNS env var is ", envSecretArns)
    secretArns := strings.Split(envSecretArns, ",")
    for _, secretArn := range secretArns {
        klog.Info("Processing:", secretArn)
        if arn.IsARN(secretArn) {
            arnobj, _ := arn.Parse(secretArn)
            AWSRegion = arnobj.Region
        } else {
            klog.Error("Invalid ARN:", secretArn)
            continue
        }

        sess := session.Must(session.NewSession())
        svc := secretsmanager.New(sess, &aws.Config{
            Region: aws.String(AWSRegion),
        })

        input := &secretsmanager.GetSecretValueInput{
            SecretId:     aws.String(secretArn),
            VersionStage: aws.String("AWSCURRENT"),
        }

        result, err := svc.GetSecretValue(input)
        if err != nil {
            if aerr, ok := err.(awserr.Error); ok {
                switch aerr.Code() {
                case secretsmanager.ErrCodeResourceNotFoundException:
                    klog.Error(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
                case secretsmanager.ErrCodeInvalidParameterException:
                    klog.Error(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())
                case secretsmanager.ErrCodeInvalidRequestException:
                    klog.Error(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())
                case secretsmanager.ErrCodeDecryptionFailure:
                    klog.Error(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())
                case secretsmanager.ErrCodeInternalServiceError:
                    klog.Error(secretsmanager.ErrCodeInternalServiceError, aerr.Error())
                default:
                    klog.Error(aerr.Error())
                }
            } else {
                klog.Error(err)
            }
            continue
        }
        // Decrypts secret using the associated KMS CMK.
        // Depending on whether the secret is a string or binary, one of these fields will be populated.
        //var decodedBinarySecret string
        if result.SecretString != nil {
            writeStringOutput(*result.Name, *result.SecretString)
        } else {
            writeBinaryOutput(*result.Name, result.SecretBinary)
        }
        klog.Info("Done processing: ", secretArn)
    }
}

func writeStringOutput(name string, output string) {
    klog.Info("Writing data from SecretString")
    f, err := os.Create(fmt.Sprintf("/injected-secrets/%s", name))
    if err != nil {
        klog.Error(err)
        return
    }
    defer f.Close()
    len, err := f.WriteString(output)
    if err != nil {
        klog.Error(err)
        return
    } else {
        klog.Info(fmt.Sprintf("Wrote %d bytes to /injected-secrets/%s", len, name))
    }
}

func writeBinaryOutput(name string, output []byte) {
    klog.Info("Writing data from SecretBinary")
    f, err := os.Create(fmt.Sprintf("/injected-secrets/%s", name))
    if err != nil {
        klog.Error(err)
        return
    }
    defer f.Close()
    len, err := f.Write(output)
    if err != nil {
        klog.Error(err)
        return
    } else {
        klog.Info(fmt.Sprintf("Wrote %d bytes to /injected-secrets/%s", len, name))
    }
}
